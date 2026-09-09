# Темы DeepCoolGo

[English](README.md) • [Русский](README.ru.md)

**Тема** превращает одну выборку `CPUData` в кадр 320×240 для экрана DeepCool.
Поставляется как **Go-плагин** (`.so`) с единственной экспортированной функцией
`Screen`.

Подробный гайд по модели отрисовки, примитивам, хелперам, тестированию без железа и
типичным ошибкам — [CREATING_THEMES.md](CREATING_THEMES.md). Ниже — короткая
инструкция по сборке и запуску.

## Как выбирается тема

`ThemeLoaderService.Load(path)` загружает Go-плагин и возвращает `themes.Theme`.
Демон выбирает плагин при запуске из `config.yml`:

```yaml
theme: aether.so                 # имя файла внутри themes_path
themes_path: /usr/local/lib/deepcoolgo/themes
```

`config.yml` лежит в `$XDG_CONFIG_HOME/DCGO/config.yml` (см. верхнеуровневый
[README](../README.ru.md#конфигурация)).

## Сборка

Выполняй команды из корня проекта. Нужны Go, C-компилятор, `pkg-config` и заголовки
`libusb-1.0`, которые использует пакет `gousb`.

Самый простой путь — `make` из корня репозитория собирает демон и **все** плагины
тем в `build/` и `build/themes/`:

```sh
make
```

`make dev` идёт дальше: он ещё и копирует собранные плагины в `~/.config/DCGO/Themes`
и пишет `~/.config/DCGO/config.yml` с указанием на этот путь — после чего
`sudo ./build/deepcoolgo` запускается сразу.

Собрать вручную:

```sh
mkdir -p build/themes
CGO_ENABLED=1 go build -buildvcs=false -o build/deepcoolgo .
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/aether.so ./plugins/aether
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/lm360.so  ./plugins/lm360
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/stats.so  ./plugins/stats
```

`MaxTemp` и `MaxWatts` для шкал Stats задаются в
[../plugins/stats/main.go](../plugins/stats/main.go).

## Новая тема

Скопируй `plugins/custom` в отдельную папку:

```sh
cp -r plugins/custom plugins/my_theme
```

Измени отрисовку в `plugins/my_theme/main.go`. Пакет должен называться `main`. Тип
темы реализует метод `Screen`, а отдельная экспортируемая функция `Screen` передаёт
ему данные — загрузчик ищет именно эту функцию:

```go
func Screen(data themes.CPUData) []deepcool.FrameObject {
	return CustomTheme{}.Screen(data)
}
```

Используй `themes.CPUData` и `deepcool.FrameObject` из пакетов проекта. В `CPU`
приходит загрузка в процентах от 0 до 100, в `Power` — ватты, в `GHz` — гигагерцы,
в `Temp` — градусы Цельсия.

После изменений только в новой теме пересобери её `.so`, укажи её в `config.yml` и
перезапусти демон:

```sh
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/my_theme.so ./plugins/my_theme
# пропиши  theme: my_theme.so  в config.yml, затем:
sudo ./build/deepcoolgo
```

## Совместимость плагинов

Демон и плагины должны собираться для одинаковых ОС и архитектуры, одной версией Go,
с согласованными флагами, build tags и настройками окружения. Код общих пакетов и их
зависимостей должен совпадать — сохраняй исходники и `go.mod`/`go.sum`, использованные
при сборке демона. Если меняются `themes`, `DeepCool`, `System` или другая общая
зависимость — пересобери демон **и** плагины. Отладочные флаги вроде
`-gcflags='all=-N -l'` тоже должны совпадать.

Go-плагины поддерживаются на Linux, FreeBSD и macOS при включённом CGO. Загруженный
плагин нельзя выгрузить; повторный `Load` того же файла возвращает уже загруженный
код. Чтобы применить новую сборку темы — перезапусти демон, автоматического слежения
за файлами нет. Подробнее: [документация Go plugin](https://pkg.go.dev/plugin).
