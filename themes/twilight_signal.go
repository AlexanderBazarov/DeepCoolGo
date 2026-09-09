package themes

import (
	"embed"
	"fmt"
	"image/png"
	"math"
	"strconv"

	deepcool "Lm360Go/deep_cool"
)

type TwilightSignalTheme struct{}

var _ Theme = TwilightSignalTheme{}

//go:embed assets/twilight_signal.png
var twilightSignalFiles embed.FS

type signalPoint struct{ x, y float64 }
type signalBox struct{ x, y, w, h int }

var (
	signalWhite   = deepcool.RGB(247, 245, 242)
	signalMuted   = deepcool.RGB(190, 139, 233)
	signalViolet  = deepcool.RGB(143, 57, 246)
	signalMagenta = deepcool.RGB(225, 61, 205)
	signalCoral   = deepcool.RGB(255, 79, 123)
	signalDark    = deepcool.RGB(48, 26, 62)
	signalSizes   = [...]int{62, 56, 50, 44, 38, 34, 30, 26, 22, 18, 14, 11}
	signalData    = prepareTwilightSignal()
)

const signalCharacters = "0123456789-."

type signalGlyph struct {
	object deepcool.FrameObject
	w, h   int
}

type signalFace struct {
	size         int
	digitW, capH int
	glyphs       [12]signalGlyph
	units        [4]signalGlyph // W, GHz, degrees C, percent
}

type twilightSignalResources struct {
	background [deepcool.Width * deepcool.Height]deepcool.Color
	faces      []signalFace
	labels     [4]deepcool.FrameObject
	assetError error
}

func (TwilightSignalTheme) Validate() error { return signalData.assetError }

func prepareTwilightSignal() *twilightSignalResources {
	r := &twilightSignalResources{}
	r.assetError = readTwilightSignalBackground(&r.background)
	for _, size := range signalSizes {
		face := signalFace{size: size}
		for i, c := range signalCharacters {
			face.glyphs[i] = makeSignalGlyph(string(c), size)
			if i < 10 {
				face.digitW = max(face.digitW, face.glyphs[i].w)
				face.capH = max(face.capH, face.glyphs[i].h)
			}
		}
		for i, unit := range [...]string{"W", "GHz", "°C", "%"} {
			face.units[i] = makeSignalGlyph(unit, max(9, size*3/5))
		}
		r.faces = append(r.faces, face)
	}
	for i, label := range [...]struct {
		text string
		x, y int
	}{
		{"POWER", 16, 13},
		{"FREQ", 224, 14},
		{"CPU LOAD", 16, 174},
		{"TEMP", 139, 174},
	} {
		r.labels[i] = deepcool.Text(label.text, deepcool.Coords{X: label.x, Y: label.y}, deepcool.FontSansBold, 11, signalMuted)
	}
	return r
}

func makeSignalGlyph(s string, size int) signalGlyph {
	w, h := deepcool.MeasureText(s, deepcool.FontSansBold, float64(size))
	return signalGlyph{
		object: deepcool.Text(s, deepcool.Coords{}, deepcool.FontSansBold, float64(size), signalWhite),
		w:      w,
		h:      h,
	}
}

func readTwilightSignalBackground(dst *[deepcool.Width * deepcool.Height]deepcool.Color) error {
	background := deepcool.RGB(18, 12, 24)
	for i := range dst {
		dst[i] = background
	}
	f, err := twilightSignalFiles.Open("assets/twilight_signal.png")
	if err != nil {
		return fmt.Errorf("twilight signal artwork: %w", err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("twilight signal PNG: %w", err)
	}
	b := src.Bounds()
	if b.Empty() || b.Dx()*deepcool.Height != b.Dy()*deepcool.Width {
		return fmt.Errorf("twilight signal artwork must be 4:3; got %dx%d", b.Dx(), b.Dy())
	}
	const samples = 4
	for y := 0; y < deepcool.Height; y++ {
		for x := 0; x < deepcool.Width; x++ {
			var rr, gg, bb uint32
			for sy := 0; sy < samples; sy++ {
				py := b.Min.Y + ((y*samples*2+sy*2+1)*b.Dy())/(deepcool.Height*samples*2)
				for sx := 0; sx < samples; sx++ {
					px := b.Min.X + ((x*samples*2+sx*2+1)*b.Dx())/(deepcool.Width*samples*2)
					r, g, blue, _ := src.At(px, py).RGBA()
					rr, gg, bb = rr+r, gg+g, bb+blue
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

func signalClamp(value, low, high float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return low
	}
	return math.Max(low, math.Min(high, value))
}

func formatSignalValue(value float64, decimals int, nonnegative bool) string {
	if math.IsNaN(value) || math.IsInf(value, 0) || (nonnegative && value < 0) {
		return "--"
	}
	if math.Abs(value) < 0.5/math.Pow10(decimals) {
		value = 0
	}
	s := strconv.FormatFloat(value, 'f', decimals, 64)
	if len(s) > 24 {
		return "--"
	}
	return s
}

func signalGlyphIndex(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c == '-' {
		return 10
	}
	return 11
}

func signalAdvance(face *signalFace, c byte) int {
	if c >= '0' && c <= '9' {
		return face.digitW + 1
	}
	return face.glyphs[signalGlyphIndex(c)].w + 2
}

func signalNumberWidth(s string, face *signalFace) int {
	w := 0
	for i := 0; i < len(s); i++ {
		w += signalAdvance(face, s[i])
	}
	return max(0, w-1)
}

type signalGlyphPosition struct {
	glyph *signalGlyph
	x, y  int
}

func signalMetric(value string, unit int, box signalBox, maxSize int) [2]deepcool.FrameObject {
	const gap = 5
	var face *signalFace
	var numberW, groupH int
	for _, candidate := range [...]string{value, "--"} {
		for i := range signalData.faces {
			f := &signalData.faces[i]
			if f.size > maxSize {
				continue
			}
			w := signalNumberWidth(candidate, f)
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
	var positions [24]signalGlyphPosition
	x := box.x
	for i := 0; i < len(value); i++ {
		c := value[i]
		g := &face.glyphs[signalGlyphIndex(c)]
		advance := signalAdvance(face, c)
		gy := bottom - g.h
		if c == '-' {
			gy = bottom - face.capH + (face.capH-g.h)/2
		}
		positions[i] = signalGlyphPosition{glyph: g, x: x + (advance-1-g.w)/2, y: gy}
		x += advance
	}
	count := len(value)
	number := deepcool.FrameObject{
		Type: "TwilightSignal.number",
		Function: func(x, y int) deepcool.FramePixel {
			if x < box.x || x >= box.x+numberW || y < box.y || y >= box.y+box.h {
				return deepcool.FramePixel{}
			}
			for i := 0; i < count; i++ {
				p := positions[i]
				if x >= p.x && x < p.x+p.glyph.w && y >= p.y && y < p.y+p.glyph.h {
					if pixel := p.glyph.object.Function(x-p.x, y-p.y); pixel.Exists {
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
		Type: "TwilightSignal.unit",
		Function: func(x, y int) deepcool.FramePixel {
			if x < ux || x >= ux+u.w || y < uy || y >= uy+u.h {
				return deepcool.FramePixel{}
			}
			return u.object.Function(x-ux, y-uy)
		},
	}
	return [2]deepcool.FrameObject{number, unitObject}
}

func signalMix(a, b deepcool.Color, t float64) deepcool.Color {
	t = signalClamp(t, 0, 1)
	av, bv := uint16(a), uint16(b)
	ar, ag, ab := float64((av>>11)&31)*255/31, float64((av>>5)&63)*255/63, float64(av&31)*255/31
	br, bg, bb := float64((bv>>11)&31)*255/31, float64((bv>>5)&63)*255/63, float64(bv&31)*255/31
	return deepcool.RGB(uint8(ar+(br-ar)*t), uint8(ag+(bg-ag)*t), uint8(ab+(bb-ab)*t))
}

func distanceToSignalSegment(px, py float64, a, b signalPoint) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	denom := dx*dx + dy*dy
	if denom == 0 {
		return math.Hypot(px-a.x, py-a.y)
	}
	t := signalClamp(((px-a.x)*dx+(py-a.y)*dy)/denom, 0, 1)
	return math.Hypot(px-(a.x+t*dx), py-(a.y+t*dy))
}

func signalCoverage(distance, halfWidth float64) int {
	a := signalClamp(halfWidth+0.75-distance, 0, 1)
	return int(math.Round(a * 255))
}

func cpuSignal(cpu float64) deepcool.FrameObject {
	load := signalClamp(cpu, 0, 100) / 100
	amp := 9 + 38*load
	points := [...]signalPoint{
		{0, 127}, {18, 127}, {37, 127 + amp*.65}, {58, 127 - amp*.65},
		{82, 127 + amp*.35}, {105, 127}, {127, 127 - amp},
		{151, 127 + amp*.78}, {178, 127 - amp*.38}, {195, 138}, {207, 147},
	}
	return deepcool.FrameObject{
		Type: "TwilightSignal.cpu",
		Function: func(x, y int) deepcool.FramePixel {
			if x < 0 || x > 208 || y < 76 || y > 166 {
				return deepcool.FramePixel{}
			}
			px, py := float64(x)+0.5, float64(y)+0.5
			best := math.MaxFloat64
			for i := 0; i < len(points)-1; i++ {
				best = math.Min(best, distanceToSignalSegment(px, py, points[i], points[i+1]))
			}
			a := signalCoverage(best, 2.3)
			if a == 0 {
				return deepcool.FramePixel{}
			}
			t := float64(x) / 208
			color := signalMix(signalViolet, signalMagenta, t*2)
			if t > 0.5 {
				color = signalMix(signalMagenta, signalCoral, (t-0.5)*2)
			}
			return deepcool.FramePixel{Exists: true, Color: color, Alpha: a}
		},
	}
}

func signalBaseline() deepcool.FrameObject {
	return deepcool.FrameObject{
		Type: "TwilightSignal.baseline",
		Function: func(x, y int) deepcool.FramePixel {
			if x < 0 || x > 208 || y < 126 || y > 128 {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: signalDark, Alpha: 180}
		},
	}
}

func signalTail() []deepcool.FrameObject {
	return []deepcool.FrameObject{
		deepcool.Rectangle(deepcool.Coords{X: 288, Y: 145}, deepcool.Coords{X: 307, Y: 149}, 0, signalCoral),
		deepcool.Rectangle(deepcool.Coords{X: 308, Y: 138}, deepcool.Coords{X: 311, Y: 156}, 0, signalCoral),
		deepcool.Rectangle(deepcool.Coords{X: 301, Y: 145}, deepcool.Coords{X: 318, Y: 149}, 0, signalCoral),
	}
}

func (TwilightSignalTheme) Screen(data CPUData) []deepcool.FrameObject {
	objects := make([]deepcool.FrameObject, 0, 17)
	objects = append(objects, deepcool.FrameObject{
		Type: "TwilightSignal.background",
		Function: func(x, y int) deepcool.FramePixel {
			if x < 0 || y < 0 || x >= deepcool.Width || y >= deepcool.Height {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: signalData.background[y*deepcool.Width+x], Alpha: 255}
		},
	})
	objects = append(objects, signalData.labels[:]...)
	objects = append(objects, signalBaseline(), cpuSignal(data.CPU))
	objects = append(objects, signalTail()...)
	power := signalMetric(formatSignalValue(data.Power, 0, true), 0, signalBox{16, 31, 187, 58}, 62)
	freq := signalMetric(formatSignalValue(data.GHz, 1, true), 1, signalBox{224, 31, 88, 32}, 30)
	load := signalMetric(formatSignalValue(data.CPU, 0, true), 3, signalBox{16, 190, 106, 34}, 34)
	temp := signalMetric(formatSignalValue(data.Temp, 0, false), 2, signalBox{139, 190, 77, 34}, 34)
	objects = append(objects, power[:]...)
	objects = append(objects, freq[:]...)
	objects = append(objects, load[:]...)
	objects = append(objects, temp[:]...)
	return objects
}
