# DeepCoolGo themes

[English](README.md) • [Русский](README.ru.md)

A **theme** turns one `CPUData` sample into a 320×240 frame for the DeepCool
display. It ships as a **Go plugin** (`.so`) exporting a single function `Screen`.

The full model — pipeline, drawing primitives, helpers, testing without hardware
and the common pitfalls — is in [CREATING_THEMES.md](CREATING_THEMES.md). This
page is the short build-and-run guide.

## How selection works

`ThemeLoaderService.Load(path)` loads a Go plugin and returns a `themes.Theme`.
The daemon picks the plugin at startup from `config.yml`:

```yaml
theme: aether.so                 # file name inside themes_path
themes_path: /usr/local/lib/deepcoolgo/themes
```

`config.yml` lives at `$XDG_CONFIG_HOME/DCGO/config.yml` (see the top-level
[README](../README.md#configuration)).

## Building

Run every command from the project root. You need Go, a C compiler, `pkg-config`
and the `libusb-1.0` headers used by `gousb`.

The simplest path — `make` from the repo root builds the daemon and **all** theme
plugins into `build/` and `build/themes/`:

```sh
make
```

`make dev` goes one step further: it also copies the built plugins into
`~/.config/DCGO/Themes` and writes a `~/.config/DCGO/config.yml` pointing there, so
`sudo ./build/deepcoolgo` runs straight away.

To build things by hand:

```sh
mkdir -p build/themes
CGO_ENABLED=1 go build -buildvcs=false -o build/deepcoolgo .
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/aether.so ./plugins/aether
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/lm360.so  ./plugins/lm360
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/stats.so  ./plugins/stats
```

`MaxTemp` and `MaxWatts` for the Stats gauges are set in
[../plugins/stats/main.go](../plugins/stats/main.go).

## Adding a theme

Copy `plugins/custom` into its own folder:

```sh
cp -r plugins/custom plugins/my_theme
```

Edit the drawing in `plugins/my_theme/main.go`. The package must be `main`. A theme
type implements the `Screen` method, and a separate exported `Screen` function
hands data to it — the loader looks up exactly this function:

```go
func Screen(data themes.CPUData) []deepcool.FrameObject {
	return CustomTheme{}.Screen(data)
}
```

Use `themes.CPUData` and `deepcool.FrameObject` from the project's packages. `CPU`
is load in percent (0–100), `Power` is watts, `GHz` is gigahertz, `Temp` is
degrees Celsius.

After changing only your new theme, rebuild its `.so`, point `config.yml` at it
and restart the daemon:

```sh
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/my_theme.so ./plugins/my_theme
# set  theme: my_theme.so  in config.yml, then:
sudo ./build/deepcoolgo
```

## Plugin compatibility

The daemon and its plugins must be built for the same OS and architecture, with
the same Go version, matching build flags, build tags and environment. The source
of the shared packages and their dependencies must match — keep the sources and
the `go.mod`/`go.sum` used to build the daemon. If `themes`, `DeepCool`, `System`
or any shared dependency changes, rebuild the daemon **and** the plugins. Debug
flags such as `-gcflags='all=-N -l'` must match too.

Go plugins are supported on Linux, FreeBSD and macOS with CGO enabled. A loaded
plugin cannot be unloaded; a second `Load` of the same file returns the
already-loaded code. To apply a new theme build, restart the daemon — there is no
file watching. More: [Go plugin docs](https://pkg.go.dev/plugin).
