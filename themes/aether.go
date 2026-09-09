package themes

import (
	"fmt"
	"math"

	deepcool "Lm360Go/deep_cool"
)

type AetherTheme struct{}

const (
	aetherScreenWidth  = 320
	aetherScreenHeight = 240
)

type aetherBox struct{ x, y, w, h int }

func aetherClipBox(objs []deepcool.FrameObject, box aetherBox) []deepcool.FrameObject {
	for i := range objs {
		draw := objs[i].Function
		objs[i].Function = func(x, y int) deepcool.FramePixel {
			if draw == nil || x < box.x || y < box.y || x >= box.x+box.w || y >= box.y+box.h {
				return deepcool.FramePixel{}
			}
			return draw(x, y)
		}
	}
	return objs
}

func aetherPanel(content []deepcool.FrameObject, x0, y0, x1, y1, r float64, near, far rgb) []deepcool.FrameObject {
	objs := make([]deepcool.FrameObject, 0, len(content)+2)
	objs = append(objs, roundRectFillAA(x0, y0, x1, y1, r, aPanel.C()))
	const inset = 1.5
	cx0, cy0, cx1, cy1 := x0+inset, y0+inset, x1-inset, y1-inset
	cr := math.Max(0, r-inset)
	minX, minY := int(math.Floor(cx0)), int(math.Floor(cy0))
	maxX, maxY := int(math.Ceil(cx1)), int(math.Ceil(cy1))
	for _, obj := range content {
		draw := obj.Function
		obj.Function = func(x, y int) deepcool.FramePixel {
			if draw == nil || x < minX || y < minY || x >= maxX || y >= maxY {
				return deepcool.FramePixel{}
			}
			pixel := draw(x, y)
			if !pixel.Exists {
				return pixel
			}
			coverage := edgeCoverage(-rrSDF(float64(x)+0.5, float64(y)+0.5, cx0, cy0, cx1, cy1, cr))
			if coverage == 0 {
				return deepcool.FramePixel{}
			}
			pixel.Alpha = int((uint16(pixel.Alpha)*uint16(coverage) + 127) / 255)
			pixel.Exists = pixel.Alpha != 0
			return pixel
		}
		objs = append(objs, obj)
	}
	return append(objs, panelFrame(x0, y0, x1, y1, r, near, far))
}

func aetherFraction(v float64) float64 {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

type rgb struct{ r, g, b float64 }

func (c rgb) C() deepcool.Color { return deepcool.RGB(uint8(c.r), uint8(c.g), uint8(c.b)) }

func mix(a, b rgb, t float64) rgb {
	t = clamp01(t)
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

func grad3(t float64, c0, c1, c2 rgb) rgb {
	t = clamp01(t)
	if t < 0.5 {
		return mix(c0, c1, t*2)
	}
	return mix(c1, c2, (t-0.5)*2)
}

var (
	aCorner = rgb{4, 9, 12}
	aGlow   = rgb{11, 22, 26}
	aPanel  = rgb{8, 17, 20}
	aWhite  = rgb{246, 247, 245}
	aText   = rgb{207, 213, 212}
	aMuted  = rgb{121, 135, 139}
	aCyan   = rgb{67, 232, 235}
	aMint   = rgb{104, 239, 210}
	aLime   = rgb{181, 247, 115}
	aBorder = rgb{91, 109, 114}
	aInact  = rgb{18, 35, 40}
	aTrack  = rgb{23, 33, 37}
)

func rrSDF(px, py, x0, y0, x1, y1, r float64) float64 {
	cx, cy := (x0+x1)/2, (y0+y1)/2
	hx, hy := (x1-x0)/2, (y1-y0)/2
	if r > hx {
		r = hx
	}
	if r > hy {
		r = hy
	}
	qx := math.Abs(px-cx) - (hx - r)
	qy := math.Abs(py-cy) - (hy - r)
	ax, ay := math.Max(qx, 0), math.Max(qy, 0)
	return hypot(ax, ay) + math.Min(math.Max(qx, qy), 0) - r
}

func roundRectFillAA(x0, y0, x1, y1, r float64, col deepcool.Color) deepcool.FrameObject {
	return deepcool.FrameObject{
		Type: "RRFill",
		Function: func(x, y int) deepcool.FramePixel {
			sd := rrSDF(float64(x)+0.5, float64(y)+0.5, x0, y0, x1, y1, r)
			a := edgeCoverage(-sd)
			if a == 0 {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: col, Alpha: a}
		},
	}
}

func roundRectStrokeGrad(x0, y0, x1, y1, r, bw float64, colorAt func(px, py float64) deepcool.Color) deepcool.FrameObject {
	return deepcool.FrameObject{
		Type: "RRStroke",
		Function: func(x, y int) deepcool.FramePixel {
			px, py := float64(x)+0.5, float64(y)+0.5
			sd := rrSDF(px, py, x0, y0, x1, y1, r)
			a := edgeCoverage(bw/2 - math.Abs(sd))
			if a == 0 {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: colorAt(px, py), Alpha: a}
		},
	}
}

func panelFrame(x0, y0, x1, y1, r float64, near, far rgb) deepcool.FrameObject {
	w, h := x1-x0, y1-y0
	return roundRectStrokeGrad(x0, y0, x1, y1, r, 1.4, func(px, py float64) deepcool.Color {
		t := 0.5*((px-x0)/w) + 0.5*(1-(py-y0)/h)
		return mix(near, far, t).C()
	})
}

func trackedWidth(s string, kind deepcool.FontKind, size float64, tracking int) int {
	if s == "" {
		return 0
	}
	x := 0
	for _, r := range s {
		if r == ' ' {
			sw, _ := deepcool.MeasureText("n", kind, size)
			x += sw/2 + tracking
			continue
		}
		w, _ := deepcool.MeasureText(string(r), kind, size)
		x += w + tracking
	}
	return x - tracking
}

func trackedText(s string, pos deepcool.Coords, kind deepcool.FontKind, size float64, tracking int, col deepcool.Color) []deepcool.FrameObject {
	objs := []deepcool.FrameObject{}
	x := pos.X
	for _, r := range s {
		if r == ' ' {
			sw, _ := deepcool.MeasureText("n", kind, size)
			x += sw/2 + tracking
			continue
		}
		ch := string(r)
		objs = append(objs, deepcool.Text(ch, deepcool.Coords{X: x, Y: pos.Y}, kind, size, col))
		w, _ := deepcool.MeasureText(ch, kind, size)
		x += w + tracking
	}
	return objs
}

func aetherTextSize(s string, kind deepcool.FontKind, size float64, tracking int) (int, int) {
	if tracking == 0 {
		return deepcool.MeasureText(s, kind, size)
	}
	h := 0
	for _, r := range s {
		_, ch := deepcool.MeasureText(string(r), kind, size)
		if ch > h {
			h = ch
		}
	}
	return trackedWidth(s, kind, size, tracking), h
}

func aetherTextBox(s string, box aetherBox, kind deepcool.FontKind, preferred float64, tracking int, centered bool, col deepcool.Color) []deepcool.FrameObject {
	if s == "" || box.w <= 0 || box.h <= 0 {
		return nil
	}
	for _, candidate := range []string{s, "--"} {
		for size := preferred; size >= 6; size -= 0.5 {
			w, h := aetherTextSize(candidate, kind, size, tracking)
			if w > box.w || h > box.h {
				continue
			}
			pos := deepcool.Coords{X: box.x, Y: box.y + (box.h-h)/2}
			if centered {
				pos.X += (box.w - w) / 2
			}
			var objs []deepcool.FrameObject
			if tracking == 0 {
				objs = []deepcool.FrameObject{deepcool.Text(candidate, pos, kind, size, col)}
			} else {
				objs = trackedText(candidate, pos, kind, size, tracking, col)
			}
			return aetherClipBox(objs, box)
		}
	}
	return nil
}

func aetherPowerValue(value string, box aetherBox) []deepcool.FrameObject {
	const gap, unitSize = 8, 20
	uw, uh := deepcool.MeasureText("W", deepcool.FontSansBold, unitSize)
	for _, candidate := range []string{value, "--"} {
		for size := 62.0; size >= 6; size -= 0.5 {
			vw, vh := deepcool.MeasureText(candidate, deepcool.FontSansBold, size)
			w, h := vw+gap+uw, vh
			if uh > h {
				h = uh
			}
			if w > box.w || h > box.h {
				continue
			}
			x := box.x + (box.w-w)/2
			bottom := box.y + (box.h-h)/2 + h
			return aetherClipBox([]deepcool.FrameObject{
				deepcool.Text(candidate, deepcool.Coords{X: x, Y: bottom - vh}, deepcool.FontSansBold, size, aWhite.C()),
				deepcool.Text("W", deepcool.Coords{X: x + vw + gap, Y: bottom - uh}, deepcool.FontSansBold, unitSize, aWhite.C()),
			}, box)
		}
	}
	return nil
}

func progressBar(x, y, w, h int, pct float64, c0, c1, c2 rgb) []deepcool.FrameObject {
	if w <= 0 || h <= 0 {
		return nil
	}
	fx0, fy0 := float64(x), float64(y)
	fx1, fy1 := float64(x+w), float64(y+h)
	r := float64(h) / 2
	pct = aetherFraction(pct)
	track := roundRectFillAA(fx0, fy0, fx1, fy1, r, aTrack.C())
	if pct == 0 {
		return []deepcool.FrameObject{track}
	}
	maxX := fx0 + float64(w)*pct
	fillRadius := math.Min(r, (maxX-fx0)/2)

	fill := deepcool.FrameObject{
		Type: "BarFill",
		Function: func(px, py int) deepcool.FramePixel {
			fpx, fpy := float64(px)+0.5, float64(py)+0.5
			sd := rrSDF(fpx, fpy, fx0, fy0, maxX, fy1, fillRadius)
			a := edgeCoverage(-sd)
			trackAlpha := edgeCoverage(-rrSDF(fpx, fpy, fx0, fy0, fx1, fy1, r))
			if trackAlpha < a {
				a = trackAlpha
			}
			if a == 0 {
				return deepcool.FramePixel{}
			}
			c := grad3((fpx-fx0)/float64(w), c0, c1, c2)
			return deepcool.FramePixel{Exists: true, Color: c.C(), Alpha: a}
		},
	}
	return []deepcool.FrameObject{
		track,
		fill,
	}
}

func segmentRow(x, y, segW, segH, gap, count int, frac float64, active, inactive, first rgb) []deepcool.FrameObject {
	if segW <= 0 || segH <= 0 || gap < 0 || count <= 0 {
		return nil
	}
	n := int(math.Round(aetherFraction(frac) * float64(count)))
	objs := make([]deepcool.FrameObject, 0, count)
	for i := 0; i < count; i++ {
		sx := float64(x + i*(segW+gap))
		c := inactive
		if i < n {
			c = active
			if i == 0 {
				c = first
			}
		}
		objs = append(objs, roundRectFillAA(sx, float64(y), sx+float64(segW), float64(y+segH), 1, c.C()))
	}
	return objs
}

func iconBox(x, y int, border rgb) deepcool.FrameObject {
	return panelFrame(float64(x), float64(y), float64(x+25), float64(y+25), 5, border, mix(border, aMuted, 0.6))
}

func iconWave(x, y int, col deepcool.Color) []deepcool.FrameObject {
	fx, fy := float64(x), float64(y)
	line := []ptf{
		{fx + 4, fy + 15}, {fx + 8, fy + 15}, {fx + 11, fy + 8},
		{fx + 14, fy + 18}, {fx + 17, fy + 11}, {fx + 21, fy + 13},
	}
	return []deepcool.FrameObject{
		iconBox(x, y, aCyan),
		strokePolyline(line, 0.9, col),
	}
}

func iconThermo(x, y int, col deepcool.Color) []deepcool.FrameObject {
	fx, fy := float64(x), float64(y)
	stem := []ptf{{fx + 12, fy + 5}, {fx + 12, fy + 15}, {fx + 10, fy + 17}}
	return []deepcool.FrameObject{
		iconBox(x, y, aCyan),
		strokePolyline(stem, 0.9, col),
		discAA(fx+12, fy+19, 3, col),
	}
}

func aetherBackground() deepcool.FrameObject {
	const cx, cy = 150.0, 132.0
	maxd := hypot(cx, cy)
	return deepcool.FrameObject{
		Type: "Bg",
		Function: func(x, y int) deepcool.FramePixel {
			d := hypot(float64(x)-cx, float64(y)-cy) / maxd
			c := mix(aGlow, aCorner, d*d)
			if x < 185 {
				lift := (185 - float64(x)) / 185 * 0.12 * (1 - d)
				c = mix(c, rgb{20, 42, 46}, lift)
			}
			return deepcool.FramePixel{Exists: true, Color: c.C(), Alpha: 255}
		},
	}
}

func aetherHeader() []deepcool.FrameObject {
	objs := trackedText("AETHER", deepcool.Coords{X: 17, Y: 15}, deepcool.FontSansBold, 15, 3, aWhite.C())
	w := trackedWidth("AETHER", deepcool.FontSansBold, 15, 3)
	objs = append(objs, trackedText("MIN", deepcool.Coords{X: 17 + w + 7, Y: 15}, deepcool.FontSansBold, 15, 3, aCyan.C())...)
	objs = append(objs, trackedText("SIMPLE POWERFUL COOL", deepcool.Coords{X: 18, Y: 34}, deepcool.FontSans, 8, 2, aMuted.C())...)

	slash := []rgb{aBorder, aBorder, aCyan, aLime}
	for i, c := range slash {
		bx := 250.0 + float64(i)*7
		objs = append(objs, strokePolyline([]ptf{{bx, 37}, {bx + 8, 22}}, 1.7, c.C()))
	}
	return objs
}

func aetherPowerPanel(watts int, pct float64) []deepcool.FrameObject {
	const x0, y0, x1, y1 = 12, 48, 180, 204
	pct = aetherFraction(pct)
	var objs []deepcool.FrameObject

	wave := roundedPath([]ptf{{15, 92}, {55, 72}, {110, 150}, {176, 138}}, 40)
	objs = append(objs, strokePolyline(wave, 28, rgb{6, 13, 16}.C()))

	objs = append(objs, aetherPowerValue(fmt.Sprintf("%d", watts), aetherBox{26, 78, 140, 61})...)

	objs = append(objs, drawChipIcon(29, 149, aCyan.C())...)
	objs = append(objs, aetherTextBox("CPU POWER", aetherBox{53, 148, 113, 20}, deepcool.FontSans, 10, 1, false, aText.C())...)

	percentW, _ := deepcool.MeasureText("100%", deepcool.FontSans, 9)
	percentX := x1 - 12 - percentW
	objs = append(objs, progressBar(26, 177, percentX-26-8, 5, pct, aCyan, aMint, aLime)...)
	objs = append(objs, aetherTextBox(fmt.Sprintf("%d%%", int(math.Round(pct*100))),
		aetherBox{percentX, 172, percentW, 15}, deepcool.FontSans, 9, 0, true, aMuted.C())...)

	return aetherPanel(objs, x0, y0, x1, y1, 11, aCyan, aBorder)
}

func aetherFreqPanel(ghz float64) []deepcool.FrameObject {
	const x0, y0, x1, y1 = 183, 48, 301, 123
	objs := iconWave(193, 60, aCyan.C())

	objs = append(objs, strokePolyline([]ptf{{236, 62}, {236, 85}}, 0.6, aMuted.C()))

	val := fmt.Sprintf("%.1f", ghz)
	frac := (ghz - 2.0) / 4.0
	if math.IsNaN(ghz) || math.IsInf(ghz, 0) {
		val = "--"
		frac = 0
	}
	objs = append(objs, aetherTextBox(val, aetherBox{242, 55, 49, 32}, deepcool.FontSansBold, 30, 0, true, aWhite.C())...)
	objs = append(objs, aetherTextBox("CPU FREQ", aetherBox{193, 91, 51, 14}, deepcool.FontSans, 8, 0, false, aMuted.C())...)
	objs = append(objs, aetherTextBox("GHz", aetherBox{247, 91, 44, 14}, deepcool.FontSansBold, 12, 0, true, aText.C())...)

	objs = append(objs, segmentRow(195, 109, 3, 6, 3, 14, frac, aCyan, aInact, aCyan)...)

	return aetherPanel(objs, x0, y0, x1, y1, 10, aCyan, aBorder)
}

func aetherTempPanel(celsius int) []deepcool.FrameObject {
	const x0, y0, x1, y1 = 183, 127, 301, 204
	objs := iconThermo(193, 139, aMint.C())
	objs = append(objs, strokePolyline([]ptf{{236, 141}, {236, 164}}, 0.6, aMuted.C()))

	val := fmt.Sprintf("%d", celsius)
	objs = append(objs, aetherTextBox(val, aetherBox{240, 135, 51, 32}, deepcool.FontSansBold, 30, 0, true, aWhite.C())...)
	objs = append(objs, aetherTextBox("CPU TEMP", aetherBox{193, 170, 51, 14}, deepcool.FontSans, 8, 0, false, aMuted.C())...)
	objs = append(objs, aetherTextBox("°C", aetherBox{247, 170, 44, 14}, deepcool.FontSansBold, 12, 0, true, aText.C())...)

	frac := float64(celsius) / 100.0
	objs = append(objs, segmentRow(195, 188, 3, 6, 3, 14, frac, aLime, aInact, aMint)...)

	return aetherPanel(objs, x0, y0, x1, y1, 10, aCyan, aLime)
}

func (a AetherTheme) Screen(data CPUData) []deepcool.FrameObject {
	objs := []deepcool.FrameObject{aetherBackground()}
	objs = append(objs, aetherHeader()...)
	objs = append(objs, aetherPowerPanel(int(data.Power), data.CPU/100.0)...)
	objs = append(objs, aetherFreqPanel(data.GHz)...)
	objs = append(objs, aetherTempPanel(int(data.Temp))...)
	return aetherClipBox(objs, aetherBox{0, 0, aetherScreenWidth, aetherScreenHeight})
}
