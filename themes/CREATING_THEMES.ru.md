# Создание тем DeepCoolGo — подробный гайд

[English](CREATING_THEMES.md) • [Русский](CREATING_THEMES.ru.md)

Тема — это код, который на каждый тик превращает показатели CPU в картинку 320×240
для экрана DeepCool. Технически тема поставляется как **Go plugin** (`.so`) с одной
экспортированной функцией `Screen`.

Короткая инструкция по сборке и запуску — в [README.md](README.md). Этот документ
описывает всю модель целиком: конвейер, модель отрисовки, контракт темы, каталог
примитивов и хелперов, пошаговое создание, сборку, тестирование без железа и типичные
ошибки.

---

## 1. Как это работает целиком

```
main.go (composition root)
  ├─ config.Load()  →  Config{ThemesPath, Theme, RefreshRate}
  ├─ themes.ThemeLoaderService.Load(cfg.ThemePath())
  │       ├─ os.Stat            — путь существует и это обычный файл
  │       ├─ plugin.Open(path)  — загрузка .so
  │       ├─ .Lookup("Screen")  — ищет символ с именем Screen
  │       └─ приведение типа к func(themes.CPUData) []deepcool.FrameObject
  ├─ deepcool.NewDeepCoolService()
  └─ app.Runner.Run(ctx)                       // ctx отменяется по SIGINT/SIGTERM
        цикл каждые cfg.Interval():
          data ← monitor.Collector.Collect()   // power, cpu, temp, ghz
          если data == prev → пропуск
          frame := deepcool.Frame{}; frame.AddAll(theme.Screen(data))
          display.PrintFrame(frame)   → frame.render() → USB
```

Ключевые факты:

- Приложение **всегда** загружает плагин — встроенной темы-фолбэка нет. Если плагин
  не загрузился, `main` логирует ошибку и выходит со статусом 1. Плагин выбирается
  полями `theme` / `themes_path` в `config.yml`
  ([config/config.go](../config/config.go)).
- Загрузчик ищет **функцию верхнего уровня** `Screen`, а не метод. Сигнатура должна
  быть **точно** `func(themes.CPUData) []deepcool.FrameObject`, иначе `Load` вернёт
  ошибку `Screen must be func(themes.CPUData) []deepcool.FrameObject`
  ([theme_loader_service.go:32-39](theme_loader_service.go#L32-L39)).
- `Screen` вызывается **не чаще раза в `refresh_rate` секунд** (по умолчанию `30`) и
  **только если значение `CPUData` изменилось** с прошлого тика
  ([app/runner.go](../app/runner.go)). `CPU` округляется вниз (`math.Floor`), `GHz`
  приходит с точностью до сотых — мелкий джиттер всё равно вызывает перерисовку.
- Если коллектор вернул ошибку (например, нет `scaling_cur_freq`), вся выборка
  отбрасывается, тик пропускается и `Screen` в этот раз не вызывается.

---

## 2. Модель отрисовки

### 2.1 Холст

| Параметр | Значение |
|---|---|
| Размер | `deepcool.Width` = 320, `deepcool.Height` = 240 |
| Начало координат | верхний левый угол |
| Ось X | вправо, `0 … 319` |
| Ось Y | **вниз**, `0 … 239` |
| Цвет | RGB565 (16 бит): 5 бит R, 6 бит G, 5 бит B |

### 2.2 `FrameObject` — единица отрисовки

```go
type FrameObject struct {
    Type     string                       // произвольная метка, для отладки
    Function func(x, y int) FramePixel     // вызывается для каждого пикселя
}

type FramePixel struct {
    Exists bool          // false → пиксель не трогаем
    Color  Color         // deepcool.Color (RGB565)
    Alpha  int           // 1..255; 0 трактуется как 255 (непрозрачно)
}
```

`Screen` возвращает **срез** `FrameObject`. При рендере для каждого из
320×240 = 76 800 пикселей перебираются **все** объекты по порядку, и результат
накладывается ([frame.go:36-65](../deep_cool/frame.go#L36-L65)):

- Порядок в срезе = **z-order**. Первый объект — самый нижний, последний — сверху.
- `Exists == false` → пиксель пропускается (так «прозрачность» и делается).
- `Alpha` смешивает цвет объекта с тем, что уже накоплено, по формуле
  `out = fg*a + bg*(255-a)` в пространстве RGB888, потом обратно в RGB565
  ([frame.go:67-104](../deep_cool/frame.go#L67-L104)).
- **`Alpha == 0` означает «непрозрачно», а не «прозрачно»**
  ([frame.go:53-55](../deep_cool/frame.go#L53-L55)). Чтобы пиксель был невидим —
  возвращайте `FramePixel{}` (то есть `Exists: false`).
- Стартовый фон — чёрный `0x0000`. Если тема не покрыла пиксель, он останется чёрным.

### 2.3 Антиалиасинг

Готовые фигуры дают гладкие края через «покрытие» (coverage): доля пикселя внутри
фигуры переводится в `Alpha`. Базовый помощник —
[`edgeCoverage(in float64) int`](lm360.go#L41-L50): `in` — знаковое расстояние в
пикселях (положительное = внутри), возвращает `0..255`. Свои фигуры со сглаживанием
строят так же: считаете SDF/расстояние, гоните через `edgeCoverage`, возвращаете
`FramePixel{Exists: a != 0, Color: col, Alpha: a}`.

### 2.4 Производительность

`Function` дергается 76 800 раз **на каждый объект**. `AetherTheme` создаёт сотни
объектов — это укладывается в тик длиной в несколько секунд, но держите функции
дешёвыми:

- Первым делом отсекайте по bounding box: `if x < minX || x >= maxX || y < minY || y >= maxY { return FramePixel{} }`.
- Тяжёлые вычисления (парсинг путей, `MeasureText`, растеризация) делайте **вне**
  `Function`, при построении объекта. `deepcool.Text` уже кэширует маски глифов
  по `(строка, шрифт, размер)` ([text.go:54-66](../deep_cool/text.go#L54-L66)).
- Не создавайте объект-на-пиксель. Один объект = одна фигура/линия/строка.

---

## 3. Контракт темы

### 3.1 Интерфейс

```go
// themes/theme.go
type Theme interface {
    Screen(data CPUData) []deepcool.FrameObject
}
```

Внутри приложения тема — это `Theme`. Плагин же экспортирует **свободную функцию**
`Screen` с той же сигнатурой; загрузчик оборачивает её в `pluginTheme`
([theme_loader_service.go:43-49](theme_loader_service.go#L43-L49)).

### 3.2 Входные данные

```go
// themes/data.go
type CPUData struct {
    CPU   float64 // загрузка, проценты 0..100 (округлена вниз)
    Power float64 // ватты (Intel RAPL: усреднённая мощность за интервал)
    GHz   float64 // средняя частота по ядрам, ГГц
    Temp  float64 // °C (k10temp/Tctl или coretemp/Package id 0)
}
```

Замечания по значениям:

- На **первом** тике `Power` часто `0` (монитору нужен предыдущий замер).
- `GHz` теоретически может быть `0`; защищайтесь от `NaN`/`Inf`, как это делает
  Aether ([aether.go:399-402](aether.go#L399-L402)).
- Верхних границ у `Power`/`Temp`/`GHz` нет — если рисуете шкалы, задавайте максимум
  сами (см. `StatsTheme.MaxTemp`, `MaxWatts`).

### 3.3 Что должен вернуть `Screen`

- Непустой срез (загрузчик-раннер в тестах считает пустой ответ ошибкой,
  [testdata/loader/main.go:17-20](testdata/loader/main.go#L17-L20)).
- Обычно первым объектом — фон на весь экран, иначе непокрытые пиксели будут чёрными.
- Ничего никуда не пишите, не блокируйтесь, не паникуйте: `Screen` — чистая функция
  «данные → картинка», вызывается из горячего цикла.

---

## 4. Каталог примитивов пакета `deepcool`

Всё это экспортировано и доступно из любого плагина.

### 4.1 Константы и цвет

```go
deepcool.Width, deepcool.Height          // 320, 240
deepcool.RGB(r, g, b uint8) Color         // упаковка в RGB565
deepcool.ColorBlack / White / Red / Green / Blue /
        Yellow / Cyan / Magenta / Orange / Gray
```

`Color` — это `uint16`. Распаковка компонентов (пригодится для превью-рендера):
`r5 = (c>>11)&0x1F`, `g6 = (c>>5)&0x3F`, `b5 = c&0x1F`.

### 4.2 Фигуры ([shapes.go](../deep_cool/shapes.go))

| Функция | Что рисует |
|---|---|
| `Rectangle(a, b Coords, border int, color Color)` | прямоугольник по двум углам; `border == 0` — залитый, `border > 0` — рамка толщиной `border` |
| `Circle(center Coords, radius, border int, color Color)` | диск (`border == 0`) или кольцо толщиной `border` |
| `Arc(center Coords, rInner, rOuter int, startDeg, sweepDeg float64, color Color)` | сегмент кольца между радиусами, от `startDeg` по часовой стрелке на `sweepDeg` |

Углы `Arc` в экранных координатах: **0° = восток, 90° = юг (вниз), 180° = запад,
270° = север (вверх)** ([shapes.go:49-52](../deep_cool/shapes.go#L49-L52)). `sweepDeg`
зажимается в `0..360`. Эти примитивы **без сглаживания** (жёсткий край).

### 4.3 Текст ([text.go](../deep_cool/text.go))

```go
deepcool.Text(s string, pos Coords, kind FontKind, sizePx float64, col Color) FrameObject
deepcool.TextCentered(s string, center Coords, kind FontKind, sizePx float64, col Color) FrameObject
deepcool.MeasureText(s string, kind FontKind, sizePx float64) (w, h int)
```

- `pos` в `Text` — **левый верхний угол** маски глифов (не базовая линия).
- `FontKind`: `FontSans`, `FontSansBold`, `FontMono`, `FontMonoBold`
  (Go fonts, `DPI: 72`, hinting off; `sizePx` — это и есть высота в пикселях).
- Текст сглажен (альфа берётся из маски глифа).
- `MeasureText` кэшируется; ей можно измерять до вёрстки (подбор размера под бокс).

### 4.4 Свой примитив

Никто не мешает вернуть собственный `FrameObject`:

```go
func vignette() deepcool.FrameObject {
    return deepcool.FrameObject{
        Type: "vignette",
        Function: func(x, y int) deepcool.FramePixel {
            dx, dy := float64(x-160), float64(y-120)
            d := math.Hypot(dx, dy) / 200
            a := int(math.Min(1, d*d) * 180)
            if a == 0 {
                return deepcool.FramePixel{}
            }
            return deepcool.FramePixel{Exists: true, Color: deepcool.ColorBlack, Alpha: a}
        },
    }
}
```

---

## 5. Хелперы пакета `themes` (переиспользуемые куски)

В файлах [lm360.go](lm360.go), [aether.go](aether.go), [dashboard.go](dashboard.go)
уже написано много полезного: сглаженные скруглённые прямоугольники, полилинии,
градиенты, прогресс-бары, сегментные шкалы, иконка чипа, круговой gauge.

**Важно про видимость:**

| Где вы пишете тему | Что доступно |
|---|---|
| файл **внутри пакета `themes/`** | всё, включая приватные `edgeCoverage`, `strokePolyline`, `roundRectAA`, `discAA`, `roundedPath`, `roundRectFillAA`, `rrSDF`, `progressBar`, `segmentRow`, `drawChipIcon`, `mix`, `grad3`, `clamp01`, `hypot`, `lerp`, `norm`, тип `rgb`, … |
| файл в **`plugins/<name>/main.go`** (`package main`) | только экспортированное: `deepcool.*`, `themes.CPUData`, `themes.Theme`, `themes.Gauge`, `themes.AetherTheme` / `LM360Theme` / `StatsTheme` |

Приватные хелперы **из `plugins/` не видны**. Поэтому рекомендованный путь —
писать логику темы файлом в пакете `themes/`, а в `plugins/` держать тонкую обёртку
(так и сделаны `plugins/aether`, `plugins/lm360`, `plugins/stats`). Если хотите
единый самодостаточный файл в `plugins/` — скопируйте нужные хелперы к себе.

Экспортированный из пакета помощник только один — `themes.Gauge(...)`
([dashboard.go:45](dashboard.go#L45)): круговая шкала с заголовком, значением,
единицей и палитрой-функцией `func(frac float64) deepcool.Color`.

---

## 6. Пошаговое создание темы

### Вариант A (рекомендуемый): логика в `themes/`, обёртка в `plugins/`

**Шаг 1.** Новый файл `themes/nebula.go`:

```go
package themes

import (
    deepcool "Lm360Go/deep_cool"
    "fmt"
    "math"
)

type NebulaTheme struct {
    MaxWatts float64
}

func (t NebulaTheme) Screen(data CPUData) []deepcool.FrameObject {
    max := t.MaxWatts
    if max <= 0 {
        max = 200
    }

    objs := []deepcool.FrameObject{
        // фон на весь экран
        deepcool.Rectangle(deepcool.Coords{}, deepcool.Coords{X: deepcool.Width, Y: deepcool.Height}, 0, deepcool.RGB(6, 8, 16)),
    }

    // заголовок
    objs = append(objs, deepcool.TextCentered("NEBULA", deepcool.Coords{X: deepcool.Width / 2, Y: 22}, deepcool.FontSansBold, 18, deepcool.RGB(120, 200, 255)))

    // приватные хелперы пакета доступны здесь:
    objs = append(objs, progressBar(24, 60, 272, 10, data.CPU/100.0, aCyan, aMint, aLime)...)
    objs = append(objs, drawChipIcon(24, 90, deepcool.RGB(120, 200, 255))...)

    watts := data.Power
    if math.IsNaN(watts) || math.IsInf(watts, 0) {
        watts = 0
    }
    objs = append(objs, progressBar(24, 120, 272, 10, watts/max, aMint, aLime, aLime)...)

    objs = append(objs, deepcool.Text(fmt.Sprintf("%.0f W   %.1f GHz   %.0f C", watts, data.GHz, data.Temp),
        deepcool.Coords{X: 24, Y: 150}, deepcool.FontMono, 14, deepcool.RGB(200, 210, 220)))

    return objs
}
```

**Шаг 2.** Обёртка-плагин `plugins/nebula/main.go`:

```go
package main

import (
    deepcool "Lm360Go/deep_cool"
    "Lm360Go/themes"
)

// Именно эту функцию ищет загрузчик. Имя, сигнатура и типы — точь-в-точь.
func Screen(data themes.CPUData) []deepcool.FrameObject {
    return themes.NebulaTheme{MaxWatts: 200}.Screen(data)
}

func main() {}
```

`func main() {}` обязателен — это по-прежнему `package main`, просто без полезного
`main`. Параметры темы (лимиты шкал, палитра, флаги) задаются здесь, при
конструировании.

### Вариант B: всё в одном файле плагина

Скопируйте `plugins/custom` и правьте
([plugins/custom/main.go](../plugins/custom/main.go)):

```sh
cp -r plugins/custom plugins/nebula
```

В таком файле пакет — `main`, приватные хелперы `themes` недоступны, поэтому либо
обходитесь `deepcool.*` и `themes.Gauge`, либо переносите нужные функции в этот же
файл.

### Требования к любому плагину

- `package main`.
- Экспортированная `func Screen(themes.CPUData) []deepcool.FrameObject`.
- Пустой `func main() {}`.
- Возвращать непустой срез; первым объектом обычно фон на весь экран.

---

## 7. Сборка и запуск

Нужны Go той же версии, что собирала приложение, C-компилятор, `pkg-config` и
заголовки `libusb-1.0` (их тянет `gousb`).

Самый простой путь — из корня репозитория `make` собирает демон и **все** плагины:

```sh
make
```

Вручную:

```sh
mkdir -p build/themes

# приложение с загрузчиком (один раз, и после любых правок общих пакетов)
CGO_ENABLED=1 go build -buildvcs=false -o build/deepcoolgo .

# плагин темы
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/nebula.so ./plugins/nebula
```

Дальше укажите её в `config.yml` и запустите демон:

```yaml
theme: nebula.so
themes_path: /абсолютный/путь/к/build/themes
```

```sh
sudo ./build/deepcoolgo
```

Встроенные обёртки собираются так же: `./plugins/aether`, `./plugins/lm360`,
`./plugins/stats`.

Флаги:

- `-buildmode=plugin` — обязателен для `.so`.
- `CGO_ENABLED=1` — Go plugins требуют CGO; заодно нужен `gousb`.
- `-buildvcs=false` — чтобы сборка не зависела от наличия git-репозитория.

---

## 8. Совместимость плагинов (частые грабли)

Go plugin — жёсткий по совместимости механизм:

- Приложение и плагин должны быть собраны для **одной ОС/архитектуры**, **одной
  версией Go**, с **согласованными** флагами, build tags и переменными окружения
  (включая отладочные `-gcflags='all=-N -l'`).
- Исходники **общих пакетов** (`themes`, `DeepCool`, `System`) и их зависимостей,
  а также `go.mod`/`go.sum` должны совпадать байт-в-байт с теми, что использовались
  при сборке приложения. Иначе `plugin.Open` вернёт ошибку вроде
  *"plugin was built with a different version of package …"*.
- Меняли `themes`/`DeepCool`/`System` — **пересоберите и приложение, и все плагины**.
- Плагин **нельзя выгрузить**. Повторный `Load` того же файла отдаёт уже загруженный
  код. Чтобы применить новую сборку темы — **перезапустите приложение** (авто-слежки
  за файлом нет).
- Поддерживается на Linux, FreeBSD, macOS при включённом CGO. Документация:
  <https://pkg.go.dev/plugin>.

---

## 9. Тестирование без железа

Экран не нужен, чтобы проверить тему — `Screen` чистая.

### 9.1 Быстрый вызов из теста

По образцу [theme_loader_service_test.go](theme_loader_service_test.go): тест
собирает `.so` через `go build -buildmode=plugin`, потом запускает раннер
([testdata/loader/main.go](testdata/loader/main.go)), который грузит плагин и печатает
`objects[0].Type`. Тот же приём годится для своей темы: подать граничные `CPUData`
(`0`, `NaN` в `GHz`, огромный `Power`) и убедиться, что срез не пустой и не паникует.

Если тема живёт в пакете `themes/`, можно и совсем просто:

```go
func TestNebulaScreenNotEmpty(t *testing.T) {
    for _, d := range []themes.CPUData{
        {},
        {CPU: 100, Power: 999, GHz: 5.9, Temp: 95},
        {GHz: math.NaN()},
    } {
        if got := (themes.NebulaTheme{}).Screen(d); len(got) == 0 {
            t.Fatalf("empty frame for %+v", d)
        }
    }
}
```

### 9.2 Рендер в PNG для глазами-проверки

`Frame.render()` приватный, но `FrameObject.Function` и поля `FramePixel`
экспортированы — компоновку легко повторить. Положите в `cmd/preview/main.go`:

```go
package main

import (
    deepcool "Lm360Go/deep_cool"
    "Lm360Go/themes"
    "image"
    "image/color"
    "image/png"
    "os"
)

func main() {
    data := themes.CPUData{CPU: 42, Power: 88, GHz: 4.3, Temp: 61}
    objs := themes.NebulaTheme{MaxWatts: 200}.Screen(data)

    img := image.NewRGBA(image.Rect(0, 0, deepcool.Width, deepcool.Height))
    for y := 0; y < deepcool.Height; y++ {
        for x := 0; x < deepcool.Width; x++ {
            var r, g, b int // старт с чёрного, как в frame.render
            for _, o := range objs {
                if o.Function == nil {
                    continue
                }
                p := o.Function(x, y)
                if !p.Exists {
                    continue
                }
                a := p.Alpha
                if a <= 0 {
                    a = 255 // как в frame.go: Alpha 0 == непрозрачно
                }
                cr, cg, cb := unpack565(p.Color)
                r = (cr*a + r*(255-a)) / 255
                g = (cg*a + g*(255-a)) / 255
                b = (cb*a + b*(255-a)) / 255
            }
            img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
        }
    }

    f, err := os.Create("preview.png")
    if err != nil {
        panic(err)
    }
    defer f.Close()
    if err := png.Encode(f, img); err != nil {
        panic(err)
    }
}

func unpack565(c deepcool.Color) (r, g, b int) {
    v := uint16(c)
    r = int((v>>11)&0x1F) * 255 / 31
    g = int((v>>5)&0x3F) * 255 / 63
    b = int(v&0x1F) * 255 / 31
    return
}
```

```sh
go run ./cmd/preview && xdg-open preview.png
```

Картинка будет попиксельно совпадать с тем, что уйдёт на экран (RGB565 + тот же
порядок наложения). Меняйте `data` руками, чтобы прогнать сцены: холодный простой,
пиковая нагрузка, «прогрев» (первый тик с `Power == 0`).

---

## 10. Отладка

- `.vscode/launch.json` уже содержит конфигурации *Launch file* и *Launch Package*.
  Для запуска `main.go` нужен корректный `config.yml` с `theme`, указывающей на
  собранный плагин. Логи идут в stderr через `slog`; метрики на каждый тик пишутся
  на уровне `Debug` ([app/runner.go](../app/runner.go)) — подними уровень
  обработчика в [main.go](../main.go), чтобы их видеть.
- Плагин под отладчиком не пройдёт по совместимости, если приложение собрано без
  `-gcflags='all=-N -l'`, а отладчик — с ним. Держите флаги одинаковыми (см. §8).
- Поле `FrameObject.Type` — свободная метка; используйте её, чтобы в превью-рендере
  логировать, какой объект закрасил спорный пиксель.
- Нет экрана под рукой → §9.2, это основной цикл разработки темы.

---

## 11. Чеклист новой темы

- [ ] Логика в `themes/<name>.go`, обёртка в `plugins/<name>/main.go`
      (или всё в одном `plugins/<name>/main.go` с копиями хелперов).
- [ ] `package main`, экспортированная `func Screen(themes.CPUData) []deepcool.FrameObject`, пустой `func main() {}`.
- [ ] Первый объект — фон 320×240 (иначе дыры будут чёрными).
- [ ] Обработаны `Power == 0` (первый тик) и `NaN`/`Inf` в `GHz`.
- [ ] У шкал задан явный максимум (нет верхней границы во входных данных).
- [ ] `Function` начинается с отсечения по bounding box; тяжёлое — вне `Function`.
- [ ] Прозрачность делается через `FramePixel{}` (`Exists: false`), не `Alpha: 0`.
- [ ] Прогнано в превью-рендере на граничных `CPUData`, среза-пустышки нет.
- [ ] Сборка: `CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/<name>.so ./plugins/<name>`.
- [ ] Приложение и плагин собраны одной версией Go, с одинаковыми флагами; после
      правок общих пакетов пересобрано всё.
- [ ] Указать в `config.yml` (`theme: <name>.so`) и запустить `sudo ./build/deepcoolgo`;
      после новой сборки темы приложение перезапущено.
