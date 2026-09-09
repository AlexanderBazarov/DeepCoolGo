# Creating DeepCoolGo themes — the full guide

[English](CREATING_THEMES.md) • [Русский](CREATING_THEMES.ru.md)

A theme is code that, on every tick, turns CPU metrics into a 320×240 image for the
DeepCool display. Technically a theme is shipped as a **Go plugin** (`.so`) with a
single exported function `Screen`.

The short build-and-run guide is in [README.md](README.md). This document describes
the whole model: the pipeline, the drawing model, the theme contract, the catalogue
of primitives and helpers, step-by-step creation, building, testing without
hardware, and the common mistakes.

---

## 1. How the whole thing works

```
main.go (composition root)
  ├─ config.Load()  →  Config{ThemesPath, Theme, RefreshRate}
  ├─ themes.ThemeLoaderService.Load(cfg.ThemePath())
  │       ├─ os.Stat            — the path exists and is a regular file
  │       ├─ plugin.Open(path)  — load the .so
  │       ├─ .Lookup("Screen")  — find a symbol named Screen
  │       └─ type-assert to func(themes.CPUData) []deepcool.FrameObject
  ├─ deepcool.NewDeepCoolService()
  └─ app.Runner.Run(ctx)                       // ctx cancelled on SIGINT/SIGTERM
        loop every cfg.Interval():
          data ← monitor.Collector.Collect()   // power, cpu, temp, ghz
          if data == prev → skip
          frame := deepcool.Frame{}; frame.AddAll(theme.Screen(data))
          display.PrintFrame(frame)   → frame.render() → USB
```

Key facts:

- The daemon **always** loads a plugin — there is no built-in fallback theme. If
  the plugin cannot be loaded, `main` logs the error and exits with status 1. The
  plugin is chosen by `theme` / `themes_path` in `config.yml`
  ([config/config.go](../config/config.go)).
- The loader looks up a **top-level function** `Screen`, not a method. The
  signature must be **exactly** `func(themes.CPUData) []deepcool.FrameObject`,
  otherwise `Load` returns
  `Screen must be func(themes.CPUData) []deepcool.FrameObject`
  ([theme_loader_service.go:32-39](theme_loader_service.go#L32-L39)).
- `Screen` is called **at most once per `refresh_rate` seconds** (default `30`) and
  **only if the `CPUData` value changed** since the previous tick
  ([app/runner.go](../app/runner.go)). `CPU` is rounded down (`math.Floor`), `GHz`
  arrives with two-decimal precision — small jitter still triggers a redraw.
- If the collector returns an error (e.g. no `scaling_cur_freq`), the whole sample
  is dropped, the tick is skipped and `Screen` is not called that time.

---

## 2. The drawing model

### 2.1 Canvas

| Parameter | Value |
|---|---|
| Size | `deepcool.Width` = 320, `deepcool.Height` = 240 |
| Origin | top-left corner |
| X axis | right, `0 … 319` |
| Y axis | **down**, `0 … 239` |
| Color | RGB565 (16-bit): 5 bits R, 6 bits G, 5 bits B |

### 2.2 `FrameObject` — the unit of drawing

```go
type FrameObject struct {
    Type     string                       // arbitrary label, for debugging
    Function func(x, y int) FramePixel     // called for every pixel
}

type FramePixel struct {
    Exists bool          // false → leave the pixel untouched
    Color  Color         // deepcool.Color (RGB565)
    Alpha  int           // 1..255; 0 is treated as 255 (opaque)
}
```

`Screen` returns a **slice** of `FrameObject`. When rendering, for each of the
320×240 = 76 800 pixels **all** objects are walked in order and the result is
composited ([frame.go:36-65](../deep_cool/frame.go#L36-L65)):

- Order in the slice = **z-order**. The first object is the bottom, the last is on
  top.
- `Exists == false` → the pixel is skipped (that is how "transparency" works).
- `Alpha` blends the object's color with what is already accumulated, using
  `out = fg*a + bg*(255-a)` in RGB888 space, then back to RGB565
  ([frame.go:67-104](../deep_cool/frame.go#L67-L104)).
- **`Alpha == 0` means "opaque", not "transparent"**
  ([frame.go:53-55](../deep_cool/frame.go#L53-L55)). To make a pixel invisible,
  return `FramePixel{}` (i.e. `Exists: false`).
- The starting background is black `0x0000`. Any pixel a theme does not cover stays
  black.

### 2.3 Anti-aliasing

The ready-made shapes get smooth edges via "coverage": the fraction of a pixel
inside the shape is converted to `Alpha`. The base helper is
[`edgeCoverage(in float64) int`](lm360.go#L41-L50): `in` is a signed distance in
pixels (positive = inside), it returns `0..255`. Build your own anti-aliased
shapes the same way: compute an SDF/distance, run it through `edgeCoverage`, return
`FramePixel{Exists: a != 0, Color: col, Alpha: a}`.

### 2.4 Performance

`Function` is invoked 76 800 times **per object**. `AetherTheme` creates hundreds
of objects — that fits inside a multi-second tick, but keep the functions cheap:

- Reject by bounding box first:
  `if x < minX || x >= maxX || y < minY || y >= maxY { return FramePixel{} }`.
- Do heavy work (path parsing, `MeasureText`, rasterisation) **outside**
  `Function`, while building the object. `deepcool.Text` already caches glyph masks
  by `(string, font, size)` ([text.go:54-66](../deep_cool/text.go#L54-L66)).
- Do not create an object per pixel. One object = one shape/line/string.

---

## 3. The theme contract

### 3.1 Interface

```go
// themes/theme.go
type Theme interface {
    Screen(data CPUData) []deepcool.FrameObject
}
```

Inside the app a theme is a `Theme`. The plugin exports a **free function**
`Screen` with the same signature; the loader wraps it in `pluginTheme`
([theme_loader_service.go:43-49](theme_loader_service.go#L43-L49)).

### 3.2 Input data

```go
// themes/data.go
type CPUData struct {
    CPU   float64 // load, percent 0..100 (rounded down)
    Power float64 // watts (Intel RAPL: average power over the interval)
    GHz   float64 // average frequency across cores, GHz
    Temp  float64 // °C (k10temp/Tctl or coretemp/Package id 0)
}
```

Notes on the values:

- On the **first** tick `Power` is often `0` (the monitor needs a previous
  reading).
- `GHz` can in theory be `0`; guard against `NaN`/`Inf` the way Aether does
  ([aether.go:399-402](aether.go#L399-L402)).
- `Power`/`Temp`/`GHz` have no upper bound — if you draw gauges, pick the maximum
  yourself (see `StatsTheme.MaxTemp`, `MaxWatts`).

### 3.3 What `Screen` must return

- A non-empty slice (the test loader-runner treats an empty response as an error,
  [testdata/loader/main.go:17-20](testdata/loader/main.go#L17-L20)).
- Usually a full-screen background as the first object, otherwise uncovered pixels
  are black.
- Do not write anywhere, do not block, do not panic: `Screen` is a pure
  "data → image" function called from a hot loop.

---

## 4. Catalogue of `deepcool` package primitives

All of this is exported and available from any plugin.

### 4.1 Constants and color

```go
deepcool.Width, deepcool.Height          // 320, 240
deepcool.RGB(r, g, b uint8) Color         // pack into RGB565
deepcool.ColorBlack / White / Red / Green / Blue /
        Yellow / Cyan / Magenta / Orange / Gray
```

`Color` is a `uint16`. Unpacking the components (handy for a preview render):
`r5 = (c>>11)&0x1F`, `g6 = (c>>5)&0x3F`, `b5 = c&0x1F`.

### 4.2 Shapes ([shapes.go](../deep_cool/shapes.go))

| Function | What it draws |
|---|---|
| `Rectangle(a, b Coords, border int, color Color)` | rectangle from two corners; `border == 0` — filled, `border > 0` — outline of thickness `border` |
| `Circle(center Coords, radius, border int, color Color)` | disc (`border == 0`) or ring of thickness `border` |
| `Arc(center Coords, rInner, rOuter int, startDeg, sweepDeg float64, color Color)` | ring segment between two radii, from `startDeg` clockwise by `sweepDeg` |

`Arc` angles in screen coordinates: **0° = east, 90° = south (down), 180° = west,
270° = north (up)** ([shapes.go:49-52](../deep_cool/shapes.go#L49-L52)). `sweepDeg`
is clamped to `0..360`. These primitives are **not anti-aliased** (hard edge).

### 4.3 Text ([text.go](../deep_cool/text.go))

```go
deepcool.Text(s string, pos Coords, kind FontKind, sizePx float64, col Color) FrameObject
deepcool.TextCentered(s string, center Coords, kind FontKind, sizePx float64, col Color) FrameObject
deepcool.MeasureText(s string, kind FontKind, sizePx float64) (w, h int)
```

- `pos` in `Text` is the **top-left corner** of the glyph mask (not the baseline).
- `FontKind`: `FontSans`, `FontSansBold`, `FontMono`, `FontMonoBold`
  (Go fonts, `DPI: 72`, hinting off; `sizePx` is the height in pixels).
- Text is anti-aliased (alpha comes from the glyph mask).
- `MeasureText` is cached; you can measure before layout (fitting a size to a box).

### 4.4 Your own primitive

Nothing stops you from returning your own `FrameObject`:

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

## 5. `themes` package helpers (reusable pieces)

The files [lm360.go](lm360.go), [aether.go](aether.go), [dashboard.go](dashboard.go)
already contain a lot of useful code: anti-aliased rounded rectangles, polylines,
gradients, progress bars, segmented gauges, a chip icon, a radial gauge.

**Important about visibility:**

| Where you write the theme | What is available |
|---|---|
| a file **inside package `themes/`** | everything, including the private `edgeCoverage`, `strokePolyline`, `roundRectAA`, `discAA`, `roundedPath`, `roundRectFillAA`, `rrSDF`, `progressBar`, `segmentRow`, `drawChipIcon`, `mix`, `grad3`, `clamp01`, `hypot`, `lerp`, `norm`, the `rgb` type, … |
| a file in **`plugins/<name>/main.go`** (`package main`) | only the exported surface: `deepcool.*`, `themes.CPUData`, `themes.Theme`, `themes.Gauge`, `themes.AetherTheme` / `LM360Theme` / `StatsTheme` |

Private helpers **are not visible from `plugins/`**. So the recommended path is to
put the theme logic in a file in package `themes/` and keep a thin wrapper in
`plugins/` (that is how `plugins/aether`, `plugins/lm360`, `plugins/stats` are
built). If you want a single self-contained file in `plugins/`, copy the helpers
you need into it.

The only helper exported from the package is `themes.Gauge(...)`
([dashboard.go:45](dashboard.go#L45)): a radial gauge with a title, value, unit and
a palette function `func(frac float64) deepcool.Color`.

---

## 6. Step-by-step theme creation

### Option A (recommended): logic in `themes/`, wrapper in `plugins/`

**Step 1.** New file `themes/nebula.go`:

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
        // full-screen background
        deepcool.Rectangle(deepcool.Coords{}, deepcool.Coords{X: deepcool.Width, Y: deepcool.Height}, 0, deepcool.RGB(6, 8, 16)),
    }

    // title
    objs = append(objs, deepcool.TextCentered("NEBULA", deepcool.Coords{X: deepcool.Width / 2, Y: 22}, deepcool.FontSansBold, 18, deepcool.RGB(120, 200, 255)))

    // the package's private helpers are available here:
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

**Step 2.** Wrapper plugin `plugins/nebula/main.go`:

```go
package main

import (
    deepcool "Lm360Go/deep_cool"
    "Lm360Go/themes"
)

// This is exactly the function the loader looks for. Name, signature and types
// must match to the letter.
func Screen(data themes.CPUData) []deepcool.FrameObject {
    return themes.NebulaTheme{MaxWatts: 200}.Screen(data)
}

func main() {}
```

`func main() {}` is required — this is still `package main`, just without a useful
`main`. Theme parameters (gauge limits, palette, flags) are set here, at
construction time.

### Option B: everything in one plugin file

Copy `plugins/custom` and edit it
([plugins/custom/main.go](../plugins/custom/main.go)):

```sh
cp -r plugins/custom plugins/nebula
```

In such a file the package is `main`, the private `themes` helpers are not
available, so either stick to `deepcool.*` and `themes.Gauge`, or move the
functions you need into the same file.

### Requirements for any plugin

- `package main`.
- An exported `func Screen(themes.CPUData) []deepcool.FrameObject`.
- An empty `func main() {}`.
- Return a non-empty slice; the first object is usually a full-screen background.

---

## 7. Building and running

You need Go of the same version that built the daemon, a C compiler, `pkg-config`
and the `libusb-1.0` headers (pulled in by `gousb`).

The easy way — from the repo root, `make` builds the daemon and **every** plugin:

```sh
make
```

By hand:

```sh
mkdir -p build/themes

# the daemon (once, and after any change to the shared packages)
CGO_ENABLED=1 go build -buildvcs=false -o build/deepcoolgo .

# the theme plugin
CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/nebula.so ./plugins/nebula
```

Then point `config.yml` at it and start the daemon:

```yaml
theme: nebula.so
themes_path: /absolute/path/to/build/themes
```

```sh
sudo ./build/deepcoolgo
```

The bundled wrappers build the same way: `./plugins/aether`, `./plugins/lm360`,
`./plugins/stats`.

Flags:

- `-buildmode=plugin` — required for the `.so`.
- `CGO_ENABLED=1` — Go plugins require CGO; `gousb` needs it too.
- `-buildvcs=false` — so the build does not depend on a git repository being
  present.

---

## 8. Plugin compatibility (common rakes)

Go plugin is a strict compatibility mechanism:

- The daemon and the plugin must be built for the **same OS/arch**, the **same Go
  version**, with **matching** flags, build tags and environment variables
  (including debug `-gcflags='all=-N -l'`).
- The source of the **shared packages** (`themes`, `DeepCool`, `System`) and their
  dependencies, plus `go.mod`/`go.sum`, must match byte-for-byte those used to
  build the daemon. Otherwise `plugin.Open` returns an error like
  *"plugin was built with a different version of package …"*.
- Changed `themes`/`DeepCool`/`System`? **Rebuild both the daemon and all
  plugins.**
- A plugin **cannot be unloaded**. A repeat `Load` of the same file returns the
  already-loaded code. To apply a new theme build — **restart the daemon** (there
  is no file watching).
- Supported on Linux, FreeBSD, macOS with CGO enabled. Docs:
  <https://pkg.go.dev/plugin>.

---

## 9. Testing without hardware

You do not need a display to check a theme — `Screen` is pure.

### 9.1 A quick call from a test

Following [theme_loader_service_test.go](theme_loader_service_test.go): the test
builds a `.so` via `go build -buildmode=plugin`, then runs a runner
([testdata/loader/main.go](testdata/loader/main.go)) that loads the plugin and
prints `objects[0].Type`. The same trick works for your theme: feed it boundary
`CPUData` (`0`, `NaN` in `GHz`, a huge `Power`) and check that the slice is not
empty and does not panic.

If the theme lives in package `themes/`, it can be even simpler:

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

### 9.2 Render to PNG for an eyeball check

`Frame.render()` is private, but `FrameObject.Function` and the `FramePixel` fields
are exported — the compositing is easy to reproduce. Put this in
`cmd/preview/main.go`:

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
            var r, g, b int // start from black, as in frame.render
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
                    a = 255 // as in frame.go: Alpha 0 == opaque
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

The image is pixel-for-pixel what will go to the display (RGB565 + the same
compositing order). Change `data` by hand to run through scenes: a cold idle, a
peak load, a "warm-up" (the first tick with `Power == 0`).

---

## 10. Debugging

- `.vscode/launch.json` already has *Launch file* and *Launch Package*
  configurations. Running `main.go` needs a valid `config.yml` with a `theme` that
  points at a built plugin. Logs go to stderr via `slog`; per-tick metrics are
  logged at `Debug` level ([app/runner.go](../app/runner.go)) — raise the handler
  level in [main.go](../main.go) to see them.
- A plugin under a debugger will fail the compatibility check if the daemon is
  built without `-gcflags='all=-N -l'` while the debugger uses it. Keep the flags
  identical (see §8).
- The `FrameObject.Type` field is a free label; use it in the preview render to log
  which object painted a disputed pixel.
- No display at hand → §9.2, that is the main theme-development loop.

---

## 11. New-theme checklist

- [ ] Logic in `themes/<name>.go`, wrapper in `plugins/<name>/main.go` (or
      everything in one `plugins/<name>/main.go` with copied helpers).
- [ ] `package main`, exported `func Screen(themes.CPUData) []deepcool.FrameObject`,
      empty `func main() {}`.
- [ ] The first object is a 320×240 background (otherwise holes are black).
- [ ] `Power == 0` (first tick) and `NaN`/`Inf` in `GHz` are handled.
- [ ] Gauges have an explicit maximum (the input has no upper bound).
- [ ] `Function` starts with a bounding-box reject; heavy work is outside
      `Function`.
- [ ] Transparency uses `FramePixel{}` (`Exists: false`), not `Alpha: 0`.
- [ ] Run through the preview render on boundary `CPUData`, no empty slice.
- [ ] Build:
      `CGO_ENABLED=1 go build -buildvcs=false -buildmode=plugin -o build/themes/<name>.so ./plugins/<name>`.
- [ ] The daemon and the plugin are built with the same Go version and the same
      flags; after changes to the shared packages everything is rebuilt.
- [ ] Point `config.yml` (`theme: <name>.so`) at it and run `sudo ./build/deepcoolgo`;
      after a new theme build, restart the daemon.
