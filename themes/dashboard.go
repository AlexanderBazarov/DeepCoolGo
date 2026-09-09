package themes

import (
	"fmt"
	"math"

	deepcool "Lm360Go/deep_cool"
)

type StatsTheme struct {
	MaxTemp  int
	MaxWatts int
}

const (
	gaugeStartDeg = 135
	gaugeSweepDeg = 270
	gaugeRInner   = 52
	gaugeROuter   = 66
	gaugeTicks    = 7
)

func heatColor(frac float64) deepcool.Color {
	frac = clamp01(frac)
	if frac < 0.5 {
		return deepcool.RGB(uint8(frac*2*255), 255, 20)
	}
	return deepcool.RGB(255, uint8((1-(frac-0.5)*2)*255), 20)
}

func coolColor(frac float64) deepcool.Color {
	frac = clamp01(frac)
	return deepcool.RGB(uint8(frac*255), uint8((1-frac)*220), 255)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func Gauge(center deepcool.Coords, value, max int, title, unit string, palette func(float64) deepcool.Color) []deepcool.FrameObject {
	frac := 0.0
	if max > 0 {
		frac = clamp01(float64(value) / float64(max))
	}
	col := palette(frac)

	dimTrack := deepcool.RGB(38, 42, 52)
	dimRing := deepcool.RGB(70, 78, 92)
	labelCol := deepcool.RGB(150, 160, 175)

	objs := []deepcool.FrameObject{
		deepcool.Arc(center, gaugeROuter+2, gaugeROuter+4, gaugeStartDeg, gaugeSweepDeg, dimRing),
		deepcool.Arc(center, gaugeRInner-4, gaugeRInner-2, gaugeStartDeg, gaugeSweepDeg, dimRing),

		deepcool.Arc(center, gaugeRInner, gaugeROuter, gaugeStartDeg, gaugeSweepDeg, dimTrack),

		deepcool.Arc(center, gaugeRInner, gaugeROuter, gaugeStartDeg, gaugeSweepDeg*frac, col),
	}

	for i := 0; i <= gaugeTicks; i++ {
		deg := gaugeStartDeg + gaugeSweepDeg*float64(i)/float64(gaugeTicks)
		tc := dimRing
		if frac > 0 && deg <= gaugeStartDeg+gaugeSweepDeg*frac+0.001 {
			tc = col
		}
		objs = append(objs, deepcool.Arc(center, gaugeROuter+3, gaugeROuter+9, deg-1.4, 2.8, tc))
	}

	if frac > 0 {
		tipDeg := gaugeStartDeg + gaugeSweepDeg*frac
		tip := arcTip(center, gaugeRInner, gaugeROuter, tipDeg)
		objs = append(objs,
			deepcool.Circle(tip, 7, 0, deepcool.RGB(255, 255, 255)),
			deepcool.Circle(tip, 4, 0, col),
		)
	}

	objs = append(objs,
		deepcool.TextCentered(title, deepcool.Coords{X: center.X, Y: center.Y - 34}, deepcool.FontSansBold, 15, labelCol),
		deepcool.TextCentered(fmt.Sprintf("%d", value), deepcool.Coords{X: center.X, Y: center.Y - 2}, deepcool.FontSansBold, 46, col),
		deepcool.TextCentered(unit, deepcool.Coords{X: center.X, Y: center.Y + 30}, deepcool.FontSans, 16, labelCol),
	)
	return objs
}

func cornerBrackets(col deepcool.Color) []deepcool.FrameObject {
	const m, L, t = 8, 22, 3
	W, H := deepcool.Width, deepcool.Height
	return []deepcool.FrameObject{
		deepcool.Rectangle(deepcool.Coords{X: m, Y: m}, deepcool.Coords{X: m + L, Y: m + t}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: m, Y: m}, deepcool.Coords{X: m + t, Y: m + L}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: W - m - L, Y: m}, deepcool.Coords{X: W - m, Y: m + t}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: W - m - t, Y: m}, deepcool.Coords{X: W - m, Y: m + L}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: m, Y: H - m - t}, deepcool.Coords{X: m + L, Y: H - m}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: m, Y: H - m - L}, deepcool.Coords{X: m + t, Y: H - m}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: W - m - L, Y: H - m - t}, deepcool.Coords{X: W - m, Y: H - m}, 0, col),
		deepcool.Rectangle(deepcool.Coords{X: W - m - t, Y: H - m - L}, deepcool.Coords{X: W - m, Y: H - m}, 0, col),
	}
}

func (s StatsTheme) Screen(data CPUData) []deepcool.FrameObject {
	accent := deepcool.RGB(0, 190, 210)
	dim := deepcool.RGB(90, 100, 115)

	objs := []deepcool.FrameObject{
		deepcool.TextCentered("SYSTEM  MONITOR", deepcool.Coords{X: deepcool.Width / 2, Y: 20}, deepcool.FontSansBold, 16, accent),
		deepcool.Rectangle(deepcool.Coords{X: 46, Y: 33}, deepcool.Coords{X: deepcool.Width - 46, Y: 34}, 0, dim),
	}

	objs = append(objs, cornerBrackets(dim)...)
	objs = append(objs, Gauge(deepcool.Coords{X: 86, Y: 132}, int(data.Temp), s.MaxTemp, "CPU TEMP", "C", heatColor)...)
	objs = append(objs, Gauge(deepcool.Coords{X: 234, Y: 132}, int(data.Power), s.MaxWatts, "POWER", "W", coolColor)...)

	objs = append(objs, deepcool.TextCentered("TEST MODE", deepcool.Coords{X: deepcool.Width / 2, Y: deepcool.Height - 16}, deepcool.FontSans, 12, dim))
	return objs
}

func arcTip(center deepcool.Coords, rInner, rOuter int, deg float64) deepcool.Coords {
	r := float64(rInner+rOuter) / 2
	rad := deg * math.Pi / 180
	return deepcool.Coords{
		X: center.X + int(math.Round(r*math.Cos(rad))),
		Y: center.Y + int(math.Round(r*math.Sin(rad))),
	}
}
