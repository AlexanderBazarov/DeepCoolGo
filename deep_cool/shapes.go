package deepcool

import "math"

func Rectangle(a Coords, b Coords, border int, color Color) FrameObject {
	minX, maxX := min(a.X, b.X), max(a.X, b.X)
	minY, maxY := min(a.Y, b.Y), max(a.Y, b.Y)

	return FrameObject{
		Type: "Rectangle",
		Function: func(x, y int) FramePixel {
			if x < minX || x > maxX || y < minY || y > maxY {
				return FramePixel{}
			}
			if border > 0 &&
				x >= minX+border && x <= maxX-border &&
				y >= minY+border && y <= maxY-border {
				return FramePixel{} // hollow interior
			}
			return FramePixel{Exists: true, Color: color, Alpha: 255}
		},
	}
}

func Circle(center Coords, radius, border int, color Color) FrameObject {
	outer := radius * radius
	inner := -1
	if border > 0 && border < radius {
		inner = (radius - border) * (radius - border)
	}

	return FrameObject{
		Type: "Circle",
		Function: func(x, y int) FramePixel {
			dx := x - center.X
			dy := y - center.Y
			d := dx*dx + dy*dy
			if d > outer || d < inner {
				return FramePixel{}
			}
			return FramePixel{Exists: true, Color: color, Alpha: 255}
		},
	}
}

func Arc(center Coords, rInner, rOuter int, startDeg, sweepDeg float64, color Color) FrameObject {
	ri2 := float64(rInner * rInner)
	ro2 := float64(rOuter * rOuter)
	start := math.Mod(math.Mod(startDeg, 360)+360, 360)
	if sweepDeg < 0 {
		sweepDeg = 0
	}
	if sweepDeg > 360 {
		sweepDeg = 360
	}

	return FrameObject{
		Type: "Arc",
		Function: func(x, y int) FramePixel {
			dx := float64(x - center.X)
			dy := float64(y - center.Y)
			d2 := dx*dx + dy*dy
			if d2 < ri2 || d2 > ro2 {
				return FramePixel{}
			}
			ang := math.Atan2(dy, dx) * 180 / math.Pi
			if ang < 0 {
				ang += 360
			}
			rel := math.Mod(ang-start+360, 360)
			if rel > sweepDeg {
				return FramePixel{}
			}
			return FramePixel{Exists: true, Color: color, Alpha: 255}
		},
	}
}
