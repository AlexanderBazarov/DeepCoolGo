<div align="center">

# DeepCoolGo

**Live CPU telemetry on your DeepCool LCD cooler — rendered in Go, driven by pluggable themes.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Platform: Linux](https://img.shields.io/badge/Platform-Linux-333?logo=linux&logoColor=white)](#requirements)
[![Made with libusb](https://img.shields.io/badge/USB-libusb--1.0-6E4C13.svg)](https://libusb.info/)

English &nbsp;•&nbsp; [Русский](README.ru.md)

</div>

---

DeepCoolGo reads CPU load, package temperature, clock speed and power draw straight
from the Linux kernel and paints a 320×240 frame to the LCD on a DeepCool cooler
over raw USB. Every visual is produced by a **theme** — a Go plugin (`.so`) with a
single `Screen(data) []FrameObject` function — so you can restyle the display
without touching the daemon.

## Table of contents

- [Features](#features)
- [Requirements](#requirements)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Themes](#themes)
- [Run as a service](#run-as-a-service)
- [How it works](#how-it-works)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features

- **Zero external daemons.** Talks to the display directly through `libusb`; no
  vendor software, no root helper service beyond the unit you install.
- **Pluggable themes.** Five bundled looks, and a documented plugin contract for
  building your own — see [themes/CREATING_THEMES.md](themes/CREATING_THEMES.md).
- **Cheap on resources.** One goroutine, a ticker, and a frame is pushed only when
  a metric actually changes.
- **Software-rendered.** Anti-aliased shapes, text and gauges composited in pure
  Go — pixel-identical to a PNG preview you can render on a machine with no display.
- **One-command install.** `sudo make install` lays down the binary, the theme
  plugins, a systemd unit and a default config.

## Requirements

| | |
|---|---|
| OS | Linux (uses `/proc`, `hwmon`, `powercap`, `libusb`) |
| Toolchain | Go **1.26+**, a C compiler, `pkg-config` |
| Libraries | `libusb-1.0` development headers (pulled in by [`gousb`](https://github.com/google/gousb)) |
| Hardware | A DeepCool cooler exposing the 320×240 LCD as USB `3633:0026` |
| Privileges | Raw USB access — run as root, or add a `udev` rule (see [below](#usb-permissions)) |

Install the build dependencies:

```sh
# Fedora / Nobara / RHEL
sudo dnf install golang gcc pkgconf-pkg-config libusbx-devel

# Debian / Ubuntu
sudo apt install golang gcc pkg-config libusb-1.0-0-dev

# Arch
sudo pacman -S go gcc pkgconf libusb
```

## Quick start

```sh
git clone <this-repo> deepcoolgo && cd deepcoolgo

# build the binary + every theme plugin into ./build
make

# build, then install the bundled themes into ~/.config/DCGO/Themes and
# write ~/.config/DCGO/config.yml (no root)
make dev

# run it (needs USB access — hence sudo)
sudo ./build/deepcoolgo
```

`make dev` gives you a ready-to-run setup from source. Without it, `make` only
builds into `./build` and the daemon has no themes on its `themes_path` yet — edit
`config.yml` to point at `build/themes` or run `make dev`. For a real deployment
use `sudo make install` (see [Run as a service](#run-as-a-service)).

## Configuration

The daemon reads `config.yml` from `$XDG_CONFIG_HOME/DCGO/config.yml`. Running
manually that resolves to `~/.config/DCGO/config.yml`; the installed systemd unit
points `XDG_CONFIG_HOME` at `/etc/deepcoolgo`, so the service reads
`/etc/deepcoolgo/DCGO/config.yml`. A default file is written automatically if none
exists.

```yaml
# seconds between screen refreshes (must be > 0)
refresh_rate: 30

# theme plugin file name inside themes_path
theme: aether.so

# directory holding the compiled theme plugins (*.so)
themes_path: /usr/local/lib/deepcoolgo/themes
```

| Key | Type | Default | Meaning |
|---|---|---|---|
| `refresh_rate` | int (seconds) | `30` | How often metrics are sampled and a frame is (re)built. |
| `theme` | string | `aether.so` | Plugin file name, resolved against `themes_path`. |
| `themes_path` | string | `~/.config/DCGO/Themes` | Where `*.so` theme plugins live. `make install` uses `/usr/local/lib/deepcoolgo/themes`. |

## Themes

A theme turns one `CPUData` sample into a slice of drawing objects for the 320×240
canvas. Bundled plugins (`make plugins` builds them into `build/themes/`):

| Plugin | Description |
|---|---|
| `aether.so` | Default. Layered panels, anti-aliased gauges, soft gradients. |
| `lm360.so` | Compact readout inspired by the LM360 stock screen. |
| `stats.so` | Bar-graph dashboard; `MaxTemp` / `MaxWatts` scale limits set in [plugins/stats/main.go](plugins/stats/main.go). |
| `quietbeat.so` | Minimal pulse-style layout with embedded artwork. |
| `twilight-signal.so` | Signal-style layout with embedded artwork. |
| `custom.so` | Starting point for your own theme — copy the folder and edit. |

Switch themes by editing `theme:` in the config and restarting the daemon. Go
plugins **cannot be hot-reloaded**, and the daemon and every plugin must be built
with the same Go toolchain and matching source for the shared packages — so after
changing any of `themes/`, `deep_cool/` or `system/`, rebuild everything (`make`).

Writing a theme is fully documented in
**[themes/CREATING_THEMES.md](themes/CREATING_THEMES.md)** — pipeline, drawing
model, primitive catalogue, testing without hardware, and the plugin
compatibility rules.

## Run as a service

```sh
sudo make install      # binary + plugins + unit + default config
sudo make enable        # systemctl daemon-reload && systemctl enable --now deepcoolgo

systemctl status deepcoolgo
journalctl -u deepcoolgo -f
```

Install layout (`PREFIX` defaults to `/usr/local`, override it or `DESTDIR` for
packaging):

| Path | Contents |
|---|---|
| `/usr/local/bin/deepcoolgo` | the daemon |
| `/usr/local/lib/deepcoolgo/themes/*.so` | theme plugins |
| `/usr/lib/systemd/system/deepcoolgo.service` | systemd unit (runs as root, `Restart=on-failure`) |
| `/etc/deepcoolgo/DCGO/config.yml` | default config — **never overwritten** on reinstall |

Remove everything except the config with `sudo make uninstall`.

### USB permissions

The service runs as root because raw USB writes need `CAP_SYS_RAWIO`-level access.
To run the binary as an unprivileged user instead, add a `udev` rule:

```sh
echo 'SUBSYSTEM=="usb", ATTR{idVendor}=="3633", ATTR{idProduct}=="0026", MODE="0660", TAG+="uaccess"' \
  | sudo tee /etc/udev/rules.d/99-deepcoolgo.rules
sudo udevadm control --reload && sudo udevadm trigger
```

## How it works

```
 ┌── Monitor ──────────────┐        ┌── themes (plugin) ──┐      ┌── DeepCool ──┐
 │ /proc/stat      → CPU % │        │ Screen(CPUData)     │      │ Frame.render │
 │ hwmon           → °C    │  ───▶  │   → []FrameObject   │ ──▶  │   RGB565     │ ──▶ USB
 │ scaling_cur_freq→ GHz   │  data  │ (anti-aliased draw) │      │  libusb out  │
 │ intel-rapl      → Watts │        └─────────────────────┘      └──────────────┘
 └────────────────────────┘
        every refresh_rate seconds; a frame is sent only when data changed
```

- **Sampling** — [monitor/](monitor/): CPU utilisation from `/proc/stat` deltas,
  package temperature from `hwmon` (`k10temp`/`coretemp`), average core frequency
  from `scaling_cur_freq`, and power from the Intel RAPL energy counter
  (`/sys/class/powercap/intel-rapl:0/energy_uj`).
- **Rendering** — the theme returns `FrameObject`s; each is a `func(x, y) FramePixel`
  evaluated per pixel with painter's-algorithm compositing and alpha blending in
  RGB888, packed back to RGB565.
- **Transport** — [deep_cool/](deep_cool/): a frame header plus the pixel buffer are
  written to the display's bulk OUT endpoint via `gousb`.

## Development

```sh
make            # build binary + all plugins
make build      # binary only  → build/deepcoolgo
make plugins    # plugins only → build/themes/*.so
make dev        # build + stage themes into ~/.config/DCGO/Themes + seed config
make clean
go test ./...
```

Preview a theme without a cooler attached by rendering it to a PNG: drop the
~40-line harness from [themes/CREATING_THEMES.md §9](themes/CREATING_THEMES.md) into
`cmd/preview/main.go` and run

```sh
go run ./cmd/preview && xdg-open preview.png
```

It reproduces the same compositing math the display uses (RGB565, identical draw
order), so the image is pixel-accurate.

Project layout:

| Path | Role |
|---|---|
| [main.go](main.go) | composition root — wire everything together, handle signals |
| [app/](app/) | `Runner` — the sample/render loop |
| [config/](config/) | config struct, loading, defaults, validation |
| [monitor/](monitor/) | metric collection (`Collector` + per-metric monitors) |
| [deep_cool/](deep_cool/) | frame model, drawing primitives, USB transport |
| [system/](system/) | libusb session wrapper |
| [themes/](themes/) | theme interface, shared helpers, bundled theme logic |
| [plugins/](plugins/) | thin `package main` wrappers compiled to `.so` |
| [packaging/](packaging/) | systemd unit and default config templates |

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| `USB device 3633:0026 ... not found` | Cooler not connected, wrong model, or missing permissions — run as root or add the `udev` rule above. |
| `failed to read config` | Malformed `config.yml`; delete it to regenerate the default. |
| `plugin was built with a different version of package ...` | Daemon and theme plugin are out of sync — `make clean && make`. |
| Power always `0` | No Intel RAPL counter (AMD without `amd_energy`/`rapl`, or a VM). CPU/temp/clock still work. |
| Frame never updates | Metrics not changing (idle); lower `refresh_rate` or load the CPU to verify. |

## Contributing

Issues and pull requests are welcome. For a new theme, follow the checklist at the
end of [themes/CREATING_THEMES.md](themes/CREATING_THEMES.md) and keep the daemon
and plugins building with `make`.

## License

[MIT](LICENSE) © 2026 3x3cutable <avbazarov2006@mail.ru>
