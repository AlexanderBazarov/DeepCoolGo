package themes

import (
	"embed"
	"fmt"
	"image/png"
	"math"
	"strconv"

	deepcool "Lm360Go/deep_cool"
)

// QuietBeatTheme is the minimal Vinyl Scratch theme for the 320x240 LCD.
// CPUData.CPU is intentionally unused: the selected design has three readings.
type QuietBeatTheme struct{}

var _ Theme = QuietBeatTheme{}

//go:embed assets/quiet_beat.png
var quietBeatFiles embed.FS

var (
	quietBeatWhite = deepcool.RGB(246, 247, 245)
	quietBeatMuted = deepcool.RGB(197, 203, 207)
	quietBeatSizes = [...]int{76, 68, 60, 52, 44, 36, 28, 24, 22, 20, 16, 12}
	quietBeat      = quietBeatPrepare()
)

const quietBeatCharacters = "0123456789-."

type quietBeatBox struct{ x, y, w, h int }

type quietBeatGlyph struct {
	object deepcool.FrameObject
	w, h   int
}

type quietBeatFace struct {
	size         int
	digitW, capH int
	glyphs       [12]quietBeatGlyph
	units        [3]quietBeatGlyph // W, degrees C, GHz
}

type quietBeatResources struct {
	background [deepcool.Width * deepcool.Height]deepcool.Color
	faces      []quietBeatFace
	labels     [3]deepcool.FrameObject
	assetError error
}

// Validate reports an embedded-art error without making Screen panic or do I/O.
func (QuietBeatTheme) Validate() error { return quietBeat.assetError }

func quietBeatPrepare() *quietBeatResources {
	r := &quietBeatResources{}
	r.assetError = quietBeatReadBackground(&r.background)
	for _, size := range quietBeatSizes {
		face := quietBeatFace{size: size}
		for i, c := range quietBeatCharacters {
			face.glyphs[i] = quietBeatMakeGlyph(string(c), size)
			if i < 10 {
				face.digitW = max(face.digitW, face.glyphs[i].w)
				face.capH = max(face.capH, face.glyphs[i].h)
			}
		}
		for i, unit := range [...]string{"W", "°C", "GHz"} {
			face.units[i] = quietBeatMakeGlyph(unit, max(9, size*3/4))
		}
		r.faces = append(r.faces, face)
	}
	for i, label := range [...]struct {
		text string
		x, y int
	}{
		{"POWER", 20, 17},
		{"TEMP", 196, 129},
		{"FREQ", 196, 190},
	} {
		r.labels[i] = deepcool.Text(label.text, deepcool.Coords{X: label.x, Y: label.y}, deepcool.FontSans, 11, quietBeatMuted)
	}
	return r
}

func quietBeatMakeGlyph(s string, size int) quietBeatGlyph {
	w, h := deepcool.MeasureText(s, deepcool.FontSansBold, float64(size))
	return quietBeatGlyph{
		object: deepcool.Text(s, deepcool.Coords{}, deepcool.FontSansBold, float64(size), quietBeatWhite),
		w:      w,
		h:      h,
	}
}

// Decode and downsample once at package load. Screen only indexes RGB565 pixels.
// No PNG decoding, file access, font rasterization or growing string cache per tick.
func quietBeatReadBackground(dst *[deepcool.Width * deepcool.Height]deepcool.Color) error {
	for i := range dst {
		dst[i] = deepcool.RGB(16, 18, 19)
	}
	f, err := quietBeatFiles.Open("assets/quiet_beat.png")
	if err != nil {
		return fmt.Errorf("quiet beat artwork: %w", err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("quiet beat PNG: %w", err)
	}
	b := src.Bounds()
	if b.Empty() || b.Dx()*deepcool.Height != b.Dy()*deepcool.Width {
		return fmt.Errorf("quiet beat artwork must have a 4:3 aspect ratio; got %dx%d", b.Dx(), b.Dy())
	}
	const samples = 4
	for y := 0; y < deepcool.Height; y++ {
		for x := 0; x < deepcool.Width; x++ {
			var rr, gg, bb uint32
			for sy := 0; sy < samples; sy++ {
				py := b.Min.Y + ((y*samples*2+sy*2+1)*b.Dy())/(deepcool.Height*samples*2)
				for sx := 0; sx < samples; sx++ {
					px := b.Min.X + ((x*samples*2+sx*2+1)*b.Dx())/(deepcool.Width*samples*2)
					r, g, blue, a := src.At(px, py).RGBA()
					// RGBA returns premultiplied 16-bit channels. Flatten any alpha
					// against the same background before packing to RGB565.
					rr += r + 16*(65535-a)/255
					gg += g + 18*(65535-a)/255
					bb += blue + 19*(65535-a)/255
				}
			}
			dst[y*deepcool.Width+x] = deepcool.RGB(
				uint8((rr/(samples*samples))>>8),
				uint8((gg/(samples*samples))>>8),
				uint8((bb/(samples*samples))>>8),
			)
		}
	}
	return nil
}

func quietBeatFormat(value float64, decimals int, nonnegative bool) string {
	if math.IsNaN(value) || math.IsInf(value, 0) || (nonnegative && value < 0) {
		return "--"
	}
	// Avoid displaying -0 or -0.0 when a tiny negative value rounds to zero.
	if math.Abs(value) < 0.5/math.Pow10(decimals) {
		value = 0
	}
	s := strconv.FormatFloat(value, 'f', decimals, 64)
	if len(s) > 24 {
		return "--"
	}
	return s
}

func quietBeatGlyphIndex(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c == '-' {
		return 10
	}
	return 11 // decimal point; only quietBeatFormat output is used
}

func quietBeatAdvance(face *quietBeatFace, c byte) int {
	if c >= '0' && c <= '9' {
		return face.digitW + 1 // tabular digits, independent of the current value
	}
	return face.glyphs[quietBeatGlyphIndex(c)].w + 2
}

func quietBeatNumberWidth(s string, face *quietBeatFace) int {
	w := 0
	for i := 0; i < len(s); i++ {
		w += quietBeatAdvance(face, s[i])
	}
	return max(0, w-1)
}

type quietBeatPosition struct {
	glyph *quietBeatGlyph
	x, y  int
}

// Two objects per metric: a cached-glyph number run and a cached unit.
// Integer font sizes match the rounding in deepcool.maskKey exactly.
func quietBeatMetric(value string, unit int, box quietBeatBox, maxSize int) [2]deepcool.FrameObject {
	const gap = 7
	var face *quietBeatFace
	var numberW, groupH int
	for _, candidate := range [...]string{value, "--"} {
		for i := range quietBeat.faces {
			f := &quietBeat.faces[i]
			if f.size > maxSize {
				continue
			}
			w := quietBeatNumberWidth(candidate, f)
			h := max(f.capH, f.units[unit].h)
			if w+gap+f.units[unit].w <= box.w && h <= box.h {
				face, value, numberW, groupH = f, candidate, w, h
				break
			}
		}
		if face != nil {
			break
		}
	}
	if face == nil {
		return [2]deepcool.FrameObject{}
	}
	bottom := box.y + (box.h-groupH)/2 + groupH
	var positions [24]quietBeatPosition
	x := box.x
	for i := 0; i < len(value); i++ {
		c := value[i]
		g := &face.glyphs[quietBeatGlyphIndex(c)]
		advance := quietBeatAdvance(face, c)
		gy := bottom - g.h
		if c == '-' {
			gy = bottom - face.capH + (face.capH-g.h)/2
		}
		positions[i] = quietBeatPosition{glyph: g, x: x + (advance-1-g.w)/2, y: gy}
		x += advance
	}
	count := len(value)
	number := deepcool.FrameObject{
		Type: "QuietBeat.number",
		Function: func(x, y int) deepcool.FramePixel {
			if x < box.x || x >= box.x+numberW || y < box.y || y >= box.y+box.h {
				return deepcool.FramePixel{}
			}
			for i := 0; i < count; i++ {
				p := positions[i]
				if x >= p.x && x < p.x+p.glyph.w && y >= p.y && y < p.y+p.glyph.h {
					pixel := p.glyph.object.Function(x-p.x, y-p.y)
					if pixel.Exists {
						return pixel
					}
				}
			}
			return deepcool.FramePixel{}
		},
	}
	u := face.units[unit]
	ux, uy := box.x+numberW+gap, bottom-u.h
	unitObject := deepcool.FrameObject{
		Type: "QuietBeat.unit",
		Function: func(x, y int) deepcool.FramePixel {
			if x < ux || x >= ux+u.w || y < uy || y >= uy+u.h {
				return deepcool.FramePixel{}
			}
			return u.object.Function(x-ux, y-uy)
		},
	}
	return [2]deepcool.FrameObject{number, unitObject}
}

func (QuietBeatTheme) Screen(data CPUData) []deepcool.FrameObject {
	objs := make([]deepcool.FrameObject, 0, 10)
	objs = append(objs, deepcool.FrameObject{
		Type: "QuietBeat.background",
		Function: func(x, y int) deepcool.FramePixel {
			if x < 0 || y < 0 || x >= deepcool.Width || y >= deepcool.Height {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: quietBeat.background[y*deepcool.Width+x], Alpha: 255}
		},
	})
	objs = append(objs, quietBeat.labels[:]...)
	power := quietBeatMetric(quietBeatFormat(data.Power, 0, true), 0, quietBeatBox{20, 37, 280, 68}, 76)
	temp := quietBeatMetric(quietBeatFormat(data.Temp, 0, false), 1, quietBeatBox{196, 146, 108, 32}, 28)
	freq := quietBeatMetric(quietBeatFormat(data.GHz, 1, true), 2, quietBeatBox{196, 207, 108, 27}, 28)
	objs = append(objs, power[:]...)
	objs = append(objs, temp[:]...)
	objs = append(objs, freq[:]...)
	return objs
}
