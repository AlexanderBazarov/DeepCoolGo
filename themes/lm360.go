package themes

import (
	"fmt"
	"math"

	deepcool "Lm360Go/deep_cool"
)

type LM360Theme struct{}

var (
	lmBgCell  = deepcool.RGB(15, 15, 15)
	lmBgLine  = deepcool.RGB(4, 5, 5)
	lmValue   = deepcool.RGB(250, 250, 250)
	lmLabel   = deepcool.RGB(200, 206, 212)
	lmAccent  = deepcool.RGB(143, 237, 232)
	lmSegOff  = deepcool.RGB(90, 96, 100)
	lmRoute   = deepcool.RGB(238, 240, 240)
	lmRouteLo = deepcool.RGB(150, 154, 156)
)

type ptf struct{ x, y float64 }

func hypot(x, y float64) float64 { return math.Hypot(x, y) }

func segDist(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	wx, wy := px-ax, py-ay
	c1 := vx*wx + vy*wy
	if c1 <= 0 {
		return hypot(px-ax, py-ay)
	}
	c2 := vx*vx + vy*vy
	if c2 <= c1 {
		return hypot(px-bx, py-by)
	}
	t := c1 / c2
	return hypot(px-(ax+t*vx), py-(ay+t*vy))
}

func edgeCoverage(in float64) int {
	e := in + 0.5
	if e <= 0 {
		return 0
	}
	if e >= 1 {
		return 255
	}
	return int(e * 255)
}

func strokePolyline(pts []ptf, halfW float64, col deepcool.Color) deepcool.FrameObject {
	return deepcool.FrameObject{
		Type: "Stroke",
		Function: func(x, y int) deepcool.FramePixel {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.MaxFloat64
			for i := 0; i+1 < len(pts); i++ {
				if dd := segDist(px, py, pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y); dd < d {
					d = dd
				}
			}
			a := edgeCoverage(halfW - d)
			if a == 0 {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: col, Alpha: a}
		},
	}
}

func discAA(cx, cy, r float64, col deepcool.Color) deepcool.FrameObject {
	return deepcool.FrameObject{
		Type: "Disc",
		Function: func(x, y int) deepcool.FramePixel {
			d := hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			a := edgeCoverage(r - d)
			if a == 0 {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: col, Alpha: a}
		},
	}
}

func roundRectAA(x0, y0, x1, y1, r float64, col deepcool.Color) deepcool.FrameObject {
	cx, cy := (x0+x1)/2, (y0+y1)/2
	hx, hy := (x1-x0)/2, (y1-y0)/2
	if r > hx {
		r = hx
	}
	if r > hy {
		r = hy
	}
	return deepcool.FrameObject{
		Type: "RoundRect",
		Function: func(x, y int) deepcool.FramePixel {
			qx := math.Abs(float64(x)+0.5-cx) - (hx - r)
			qy := math.Abs(float64(y)+0.5-cy) - (hy - r)
			ax, ay := math.Max(qx, 0), math.Max(qy, 0)
			sd := hypot(ax, ay) + math.Min(math.Max(qx, qy), 0) - r
			a := edgeCoverage(-sd)
			if a == 0 {
				return deepcool.FramePixel{}
			}
			return deepcool.FramePixel{Exists: true, Color: col, Alpha: a}
		},
	}
}

func roundedPath(corners []ptf, r float64) []ptf {
	if len(corners) < 3 {
		return corners
	}
	out := []ptf{corners[0]}
	for i := 1; i+1 < len(corners); i++ {
		p0, p1, p2 := corners[i-1], corners[i], corners[i+1]
		d1x, d1y := norm(p1.x-p0.x, p1.y-p0.y)
		d2x, d2y := norm(p2.x-p1.x, p2.y-p1.y)
		t1 := ptf{p1.x - d1x*r, p1.y - d1y*r}
		t2 := ptf{p1.x + d2x*r, p1.y + d2y*r}
		out = append(out, t1)
		const steps = 6
		for k := 1; k < steps; k++ {
			tt := float64(k) / steps
			bx := lerp(lerp(t1.x, p1.x, tt), lerp(p1.x, t2.x, tt), tt)
			by := lerp(lerp(t1.y, p1.y, tt), lerp(p1.y, t2.y, tt), tt)
			out = append(out, ptf{bx, by})
		}
		out = append(out, t2)
	}
	return append(out, corners[len(corners)-1])
}

func norm(x, y float64) (float64, float64) {
	l := hypot(x, y)
	if l == 0 {
		return 0, 0
	}
	return x / l, y / l
}
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func drawBackgroundGrid() deepcool.FrameObject {
	return deepcool.FrameObject{
		Type: "Grid",
		Function: func(x, y int) deepcool.FramePixel {
			if x%10 == 0 || y%10 == 0 {
				return deepcool.FramePixel{Exists: true, Color: lmBgLine, Alpha: 255}
			}
			return deepcool.FramePixel{Exists: true, Color: lmBgCell, Alpha: 255}
		},
	}
}

func drawPanelPath() []deepcool.FrameObject {
	main := roundedPath([]ptf{
		{-2, 31}, {190, 31}, {190, 210}, {-2, 210},
	}, 12)

	right := roundedPath([]ptf{
		{188, 120}, {309, 120}, {309, -2},
	}, 11)

	stub := []ptf{{192, 133}, {214, 133}}

	return []deepcool.FrameObject{
		strokePolyline(main, 0.9, lmRoute),
		strokePolyline(right, 0.9, lmRoute),
		strokePolyline(stub, 0.8, lmRouteLo),

		roundRectAA(188, 118.5, 191.5, 122, 0.6, lmRoute),
		roundRectAA(188, 127.5, 191.5, 131, 0.6, lmRoute),
		discAA(190, 139, 1.7, lmRoute),
	}
}

func drawChipIcon(x, y int, col deepcool.Color) []deepcool.FrameObject {
	objs := []deepcool.FrameObject{
		deepcool.Rectangle(deepcool.Coords{X: x + 3, Y: y + 3}, deepcool.Coords{X: x + 11, Y: y + 11}, 1, col),
		deepcool.Rectangle(deepcool.Coords{X: x + 6, Y: y + 6}, deepcool.Coords{X: x + 8, Y: y + 8}, 0, col),
	}
	for i := 0; i < 3; i++ {
		p := i * 2
		objs = append(objs,
			deepcool.Rectangle(deepcool.Coords{X: x + 4 + p, Y: y + 1}, deepcool.Coords{X: x + 4 + p, Y: y + 2}, 0, col),
			deepcool.Rectangle(deepcool.Coords{X: x + 4 + p, Y: y + 12}, deepcool.Coords{X: x + 4 + p, Y: y + 13}, 0, col),
			deepcool.Rectangle(deepcool.Coords{X: x + 1, Y: y + 4 + p}, deepcool.Coords{X: x + 2, Y: y + 4 + p}, 0, col),
			deepcool.Rectangle(deepcool.Coords{X: x + 12, Y: y + 4 + p}, deepcool.Coords{X: x + 13, Y: y + 4 + p}, 0, col),
		)
	}
	return objs
}

func drawSegmentMeter(x, y int) []deepcool.FrameObject {
	cols := [5]deepcool.Color{lmSegOff, lmSegOff, lmAccent, lmAccent, lmAccent}
	objs := make([]deepcool.FrameObject, 0, 5)
	for i, c := range cols {
		top := float64(y + i*10)
		objs = append(objs, roundRectAA(float64(x), top, float64(x+8), top+8, 1.6, c))
	}
	return objs
}

func unitRightOf(value string, valPos deepcool.Coords, valKind deepcool.FontKind, valSize float64,
	unit string, unitKind deepcool.FontKind, unitSize float64, gap, dyTop int, col deepcool.Color) deepcool.FrameObject {
	vw, vh := deepcool.MeasureText(value, valKind, valSize)
	_, uh := deepcool.MeasureText(unit, unitKind, unitSize)
	pos := deepcool.Coords{X: valPos.X + vw + gap, Y: valPos.Y + vh - uh + dyTop}
	return deepcool.Text(unit, pos, unitKind, unitSize, col)
}

func drawPower(watts int) []deepcool.FrameObject {
	val := fmt.Sprintf("%d", watts)
	valPos := deepcool.Coords{X: 31, Y: 91}
	const valSize = 68

	objs := []deepcool.FrameObject{
		deepcool.Text(val, valPos, deepcool.FontSansBold, valSize, lmValue),
		unitRightOf(val, valPos, deepcool.FontSansBold, valSize, "W", deepcool.FontSansBold, 22, 8, -3, lmValue),
	}
	objs = append(objs, drawChipIcon(33, 153, lmAccent)...)
	objs = append(objs, deepcool.Text("CPU Power", deepcool.Coords{X: 52, Y: 155}, deepcool.FontSans, 12, lmLabel))
	return objs
}

func drawFrequency(ghz float64) []deepcool.FrameObject {
	val := fmt.Sprintf("%.1f", ghz)
	valPos := deepcool.Coords{X: 211, Y: 44}
	const valSize = 38

	objs := []deepcool.FrameObject{
		deepcool.Text(val, valPos, deepcool.FontSansBold, valSize, lmValue),
		unitRightOf(val, valPos, deepcool.FontSansBold, valSize, "GHz", deepcool.FontSansBold, 13, 6, -2, lmLabel),
	}
	objs = append(objs, drawChipIcon(213, 87, lmAccent)...)
	objs = append(objs, deepcool.Text("CPU FREQ", deepcool.Coords{X: 232, Y: 89}, deepcool.FontSans, 11, lmLabel))
	return objs
}

func drawTemperature(celsius int) []deepcool.FrameObject {
	val := fmt.Sprintf("%d", celsius)
	valPos := deepcool.Coords{X: 209, Y: 150}
	const valSize = 42

	objs := []deepcool.FrameObject{
		deepcool.Text(val, valPos, deepcool.FontSansBold, valSize, lmValue),
		unitRightOf(val, valPos, deepcool.FontSansBold, valSize, "°C", deepcool.FontSansBold, 16, 6, -14, lmValue),
	}
	objs = append(objs, drawChipIcon(212, 193, lmAccent)...)
	objs = append(objs, deepcool.Text("CPU TEMP", deepcool.Coords{X: 231, Y: 195}, deepcool.FontSans, 11, lmLabel))
	return objs
}

func (l LM360Theme) Screen(data CPUData) []deepcool.FrameObject {
	objs := []deepcool.FrameObject{drawBackgroundGrid()}
	objs = append(objs, drawPanelPath()...)
	objs = append(objs, drawSegmentMeter(180, 52)...)
	objs = append(objs, drawPower(int(data.Power))...)
	objs = append(objs, drawFrequency(data.GHz)...)
	objs = append(objs, drawTemperature(int(data.Temp))...)
	return objs
}
