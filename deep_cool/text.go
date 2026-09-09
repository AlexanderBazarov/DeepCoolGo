package deepcool

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type FontKind int

const (
	FontSans FontKind = iota
	FontSansBold
	FontMono
	FontMonoBold
)

var fontTTF = map[FontKind][]byte{
	FontSans:     goregular.TTF,
	FontSansBold: gobold.TTF,
	FontMono:     gomono.TTF,
	FontMonoBold: gomonobold.TTF,
}

var parsedFonts = map[FontKind]*opentype.Font{}

func fontFor(kind FontKind) *opentype.Font {
	if f, ok := parsedFonts[kind]; ok {
		return f
	}
	f, err := opentype.Parse(fontTTF[kind])
	if err != nil {
		panic(err)
	}
	parsedFonts[kind] = f
	return f
}

const fontSS = 4

type maskKey struct {
	s    string
	kind FontKind
	size int
}

var maskCache = map[maskKey]*image.Alpha{}

func Text(s string, pos Coords, kind FontKind, sizePx float64, col Color) FrameObject {
	if sizePx < 1 {
		sizePx = 1
	}
	key := maskKey{s: s, kind: kind, size: int(sizePx + 0.5)}
	mask, ok := maskCache[key]
	if !ok {
		mask = rasterize(s, fontFor(kind), sizePx)
		maskCache[key] = mask
	}
	dx, dy := mask.Bounds().Dx(), mask.Bounds().Dy()

	return FrameObject{
		Type: "Text",
		Function: func(x, y int) FramePixel {
			mx := x - pos.X
			my := y - pos.Y
			if mx < 0 || my < 0 || mx >= dx || my >= dy {
				return FramePixel{}
			}
			a := mask.AlphaAt(mx, my).A
			if a == 0 {
				return FramePixel{}
			}
			return FramePixel{Exists: true, Color: col, Alpha: int(a)}
		},
	}
}

func maskFor(s string, kind FontKind, sizePx float64) *image.Alpha {
	if sizePx < 1 {
		sizePx = 1
	}
	key := maskKey{s: s, kind: kind, size: int(sizePx + 0.5)}
	mask, ok := maskCache[key]
	if !ok {
		mask = rasterize(s, fontFor(kind), sizePx)
		maskCache[key] = mask
	}
	return mask
}

func MeasureText(s string, kind FontKind, sizePx float64) (w, h int) {
	b := maskFor(s, kind, sizePx).Bounds()
	return b.Dx(), b.Dy()
}

func TextCentered(s string, center Coords, kind FontKind, sizePx float64, col Color) FrameObject {
	w, h := MeasureText(s, kind, sizePx)
	return Text(s, Coords{X: center.X - w/2, Y: center.Y - h/2}, kind, sizePx, col)
}

func rasterize(s string, ttf *opentype.Font, sizePx float64) *image.Alpha {
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    sizePx,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		panic(err)
	}
	defer face.Close()

	drawer := &font.Drawer{
		Face: face,
		Src:  image.NewUniform(color.Alpha{A: 255}),
	}

	bounds, _ := drawer.BoundString(s)
	w := (bounds.Max.X - bounds.Min.X).Ceil()
	h := (bounds.Max.Y - bounds.Min.Y).Ceil()

	if w <= 0 || h <= 0 {
		return image.NewAlpha(image.Rect(0, 0, 1, 1))
	}

	img := image.NewAlpha(image.Rect(0, 0, w, h))

	drawer.Dst = img

	drawer.Dot = fixed.Point26_6{
		X: -bounds.Min.X,
		Y: -bounds.Min.Y,
	}

	drawer.DrawString(s)

	return img
}
