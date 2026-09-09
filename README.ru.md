<div align="center">

# DeepCoolGo

**CPU-телеметрия в реальном времени на LCD-экране кулера DeepCool — рендер на Go, оформление через плагины-темы.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Platform: Linux](https://img.shields.io/badge/Platform-Linux-333?logo=linux&logoColor=white)](#требования)
[![Made with libusb](https://img.shields.io/badge/USB-libusb--1.0-6E4C13.svg)](https://libusb.info/)

[English](README.md) &nbsp;•&nbsp; Русский

</div>

---

DeepCoolGo читает загрузку CPU, температуру пакета, частоту и потребляемую мощность
напрямую из ядра Linux и рисует кадр 320×240 на LCD-экране кулера DeepCool по сырому
USB. Каждая картинка формируется **темой** — Go-плагином (`.so`) с единственной
функцией `Screen(data) []FrameObject` — поэтому оформление меняется без правок самого
демона.

## Оглавление

- [Возможности](#возможности)
- [Требования](#требования)
- [Быстрый старт](#быстрый-старт)
- [Конфигурация](#конфигурация)
- [Темы](#темы)
- [Запуск как служба (systemd)](#запуск-как-служба-systemd)
- [Как это устроено](#как-это-устроено)
- [Разработка](#разработка)
- [Диагностика](#диагностика)
- [Вклад в проект](#вклад-в-проект)
- [Лицензия](#лицензия)

## Возможности

- **Никаких сторонних демонов.** Общается с экраном напрямую через `libusb`; не нужен
  фирменный софт и отдельный root-хелпер, кроме устанавливаемого юнита.
- **Плагины-темы.** Пять готовых оформлений и документированный контракт плагина для
  собственных — см. [themes/CREATING_THEMES.md](themes/CREATING_THEMES.md).
- **Дёшево по ресурсам.** Одна горутина, тикер, кадр отправляется только когда
  метрика реально изменилась.
- **Программный рендер.** Сглаженные фигуры, текст и шкалы собираются на чистом Go —
  попиксельно совпадает с PNG-превью, которое можно отрендерить на машине без экрана.
- **Установка одной командой.** `sudo make install` кладёт бинарь, плагины тем,
  systemd-юнит и конфиг по умолчанию.

## Требования

| | |
|---|---|
| ОС | Linux (используются `/proc`, `hwmon`, `powercap`, `libusb`) |
| Тулчейн | Go **1.26+**, C-компилятор, `pkg-config` |
| Библиотеки | Заголовки `libusb-1.0` (их тянет [`gousb`](https://github.com/google/gousb)) |
| Железо | Кулер DeepCool с LCD 320×240, USB `3633:0026` |
| Права | Сырой доступ к USB — запуск от root или правило `udev` (см. [ниже](#права-на-usb)) |

Установка зависимостей сборки:

```sh
# Fedora / Nobara / RHEL
sudo dnf install golang gcc pkgconf-pkg-config libusbx-devel

# Debian / Ubuntu
sudo apt install golang gcc pkg-config libusb-1.0-0-dev

# Arch
sudo pacman -S go gcc pkgconf libusb
```

## Быстрый старт

```sh
git clone <this-repo> deepcoolgo && cd deepcoolgo

# собрать бинарь + все плагины тем в ./build
make

# собрать и разложить встроенные темы в ~/.config/DCGO/Themes,
# записать ~/.config/DCGO/config.yml (без root)
make dev

# запустить (нужен доступ к USB — отсюда sudo)
sudo ./build/deepcoolgo
```

`make dev` даёт готовую к запуску конфигурацию из исходников. Без него `make` только
собирает в `./build`, и у демона ещё нет тем на `themes_path` — либо пропиши в
`config.yml` путь `build/themes`, либо выполни `make dev`. Для реального развёртывания
используй `sudo make install` (см. [Запуск как служба](#запуск-как-служба-systemd)).

## Конфигурация

Демон читает `config.yml` из `$XDG_CONFIG_HOME/DCGO/config.yml`. При ручном запуске
это `~/.config/DCGO/config.yml`; установленный systemd-юнит выставляет
`XDG_CONFIG_HOME` в `/etc/deepcoolgo`, поэтому служба читает
`/etc/deepcoolgo/DCGO/config.yml`. Если файла нет — он создаётся автоматически.

```yaml
# секунды между обновлениями экрана (должно быть > 0)
refresh_rate: 30

# имя файла плагина темы внутри themes_path
theme: aether.so

# каталог с собранными плагинами тем (*.so)
themes_path: /usr/local/lib/deepcoolgo/themes
```

| Ключ | Тип | По умолчанию | Смысл |
|---|---|---|---|
| `refresh_rate` | int (секунды) | `30` | Как часто снимаются метрики и (пере)собирается кадр. |
| `theme` | строка | `aether.so` | Имя файла плагина, ищется относительно `themes_path`. |
| `themes_path` | строка | `~/.config/DCGO/Themes` | Где лежат плагины `*.so`. `make install` использует `/usr/local/lib/deepcoolgo/themes`. |

## Темы

Тема превращает одну выборку `CPUData` в срез объектов отрисовки для холста 320×240.
Готовые плагины (`make plugins` собирает их в `build/themes/`):

| Плагин | Описание |
|---|---|
| `aether.so` | По умолчанию. Слоистые панели, сглаженные шкалы, мягкие градиенты. |
| `lm360.so` | Компактная выкладка в духе штатного экрана LM360. |
| `stats.so` | Дашборд со столбчатыми шкалами; лимиты `MaxTemp` / `MaxWatts` — в [plugins/stats/main.go](plugins/stats/main.go). |
| `quietbeat.so` | Минималистичная «пульсовая» раскладка со встроенной картинкой. |
| `twilight-signal.so` | «Сигнальная» раскладка со встроенной картинкой. |
| `custom.so` | Заготовка для своей темы — скопируйте папку и правьте. |

Смена темы — правка `theme:` в конфиге и перезапуск демона. Go-плагины **нельзя
перезагрузить на лету**, а демон и все плагины должны быть собраны одним тулчейном Go
и с одинаковыми исходниками общих пакетов — поэтому после изменений в `themes/`,
`deep_cool/` или `system/` пересобирайте всё (`make`).

Создание темы полностью описано в
**[themes/CREATING_THEMES.md](themes/CREATING_THEMES.md)** — конвейер, модель
отрисовки, каталог примитивов, тестирование без железа и правила совместимости
плагинов.

## Запуск как служба (systemd)

```sh
sudo make install      # бинарь + плагины + юнит + конфиг по умолчанию
sudo make enable        # systemctl daemon-reload && systemctl enable --now deepcoolgo

systemctl status deepcoolgo
journalctl -u deepcoolgo -f
```

Раскладка установки (`PREFIX` по умолчанию `/usr/local`; переопределяйте его или
`DESTDIR` для пакетирования):

| Путь | Содержимое |
|---|---|
| `/usr/local/bin/deepcoolgo` | демон |
| `/usr/local/lib/deepcoolgo/themes/*.so` | плагины тем |
| `/usr/lib/systemd/system/deepcoolgo.service` | systemd-юнит (от root, `Restart=on-failure`) |
| `/etc/deepcoolgo/DCGO/config.yml` | конфиг по умолчанию — **не перезаписывается** при переустановке |

Удалить всё, кроме конфига: `sudo make uninstall`.

### Права на USB

Служба работает от root, потому что сырые записи в USB требуют доступа уровня
`CAP_SYS_RAWIO`. Чтобы запускать бинарь от обычного пользователя, добавьте правило
`udev`:

```sh
echo 'SUBSYSTEM=="usb", ATTR{idVendor}=="3633", ATTR{idProduct}=="0026", MODE="0660", TAG+="uaccess"' \
  | sudo tee /etc/udev/rules.d/99-deepcoolgo.rules
sudo udevadm control --reload && sudo udevadm trigger
```

## Как это устроено

```
 ┌── Monitor ──────────────┐        ┌── themes (плагин) ──┐      ┌── DeepCool ──┐
 │ /proc/stat      → CPU % │        │ Screen(CPUData)     │      │ Frame.render │
 │ hwmon           → °C    │  ───▶  │   → []FrameObject   │ ──▶  │   RGB565     │ ──▶ USB
 │ scaling_cur_freq→ GHz   │  data  │ (сглаженный рендер) │      │  libusb out  │
 │ intel-rapl      → Вт    │        └─────────────────────┘      └──────────────┘
 └────────────────────────┘
        каждые refresh_rate секунд; кадр уходит только если данные изменились
```

- **Съём метрик** — [monitor/](monitor/): загрузка CPU по дельтам `/proc/stat`,
  температура пакета из `hwmon` (`k10temp`/`coretemp`), средняя частота ядер из
  `scaling_cur_freq`, мощность — из счётчика энергии Intel RAPL
  (`/sys/class/powercap/intel-rapl:0/energy_uj`).
- **Рендер** — тема возвращает `FrameObject`-ы; каждый — это `func(x, y) FramePixel`,
  вызываемый на каждый пиксель, с наложением «алгоритмом художника» и альфа-смешением
  в RGB888 и упаковкой обратно в RGB565.
- **Транспорт** — [deep_cool/](deep_cool/): заголовок кадра плюс буфер пикселей пишутся
  в bulk-OUT-эндпоинт экрана через `gousb`.

## Разработка

```sh
make            # собрать бинарь + все плагины
make build      # только бинарь  → build/deepcoolgo
make plugins    # только плагины → build/themes/*.so
make dev        # сборка + раскладка тем в ~/.config/DCGO/Themes + конфиг
make clean
go test ./...
```

Превью темы без подключённого кулера — рендер в PNG: положите стенд на ~40 строк
из [themes/CREATING_THEMES.md §9](themes/CREATING_THEMES.md) в `cmd/preview/main.go`
и запустите

```sh
go run ./cmd/preview && xdg-open preview.png
```

Там воспроизведена та же математика наложения, что и на экране (RGB565, тот же
порядок отрисовки), поэтому картинка попиксельно точна.

Структура проекта:

| Путь | Роль |
|---|---|
| [main.go](main.go) | composition root — сборка зависимостей, обработка сигналов |
| [app/](app/) | `Runner` — цикл «снять метрики → отрисовать» |
| [config/](config/) | структура конфига, загрузка, дефолты, валидация |
| [monitor/](monitor/) | сбор метрик (`Collector` + мониторы по метрикам) |
| [deep_cool/](deep_cool/) | модель кадра, примитивы отрисовки, USB-транспорт |
| [system/](system/) | обёртка над сессией libusb |
| [themes/](themes/) | интерфейс тем, общие хелперы, логика встроенных тем |
| [plugins/](plugins/) | тонкие обёртки `package main`, компилируемые в `.so` |
| [packaging/](packaging/) | шаблоны systemd-юнита и конфига по умолчанию |

## Диагностика

| Симптом | Вероятная причина |
|---|---|
| `USB device 3633:0026 ... not found` | Кулер не подключён, не та модель или нет прав — запустите от root или добавьте правило `udev` выше. |
| `failed to read config` | Битый `config.yml`; удалите его, чтобы пересоздать дефолт. |
| `plugin was built with a different version of package ...` | Демон и плагин темы рассинхронизированы — `make clean && make`. |
| Мощность всегда `0` | Нет счётчика Intel RAPL (AMD без `amd_energy`/`rapl` или ВМ). CPU/температура/частота работают. |
| Кадр не обновляется | Метрики не меняются (простой); уменьшите `refresh_rate` или нагрузите CPU для проверки. |

## Вклад в проект

Issue и pull request приветствуются. Для новой темы пройдите по чеклисту в конце
[themes/CREATING_THEMES.md](themes/CREATING_THEMES.md) и следите, чтобы демон и
плагины собирались через `make`.

## Лицензия

[MIT](LICENSE) © 2026, участники DeepCoolGo
