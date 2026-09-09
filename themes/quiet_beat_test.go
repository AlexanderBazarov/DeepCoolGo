package themes

import (
	"math"
	"testing"

	deepcool "Lm360Go/deep_cool"
)

func TestQuietBeatArtwork(t *testing.T) {
	if err := (QuietBeatTheme{}).Validate(); err != nil {
		t.Fatal(err)
	}
	// The asset must contain colored pony artwork in the reserved lower left.
	colored := 0
	for y := 111; y < deepcool.Height; y++ {
		for x := 0; x < 176; x++ {
			v := uint16(quietBeat.background[y*deepcool.Width+x])
			r, b := int((v>>11)&31), int(v&31)
			if b-r > 8 {
				colored++
			}
		}
	}
	if colored < 100 {
		t.Fatal("missing colored pony artwork")
	}
}

func TestQuietBeatFrames(t *testing.T) {
	theme := QuietBeatTheme{}
	cases := []CPUData{
		{},
		{CPU: 87, Power: 230, Temp: 69, GHz: 5.7},
		{CPU: 100, Power: 999, Temp: 100, GHz: 10},
		{Power: 9999, Temp: -40, GHz: 0},
		{Power: math.MaxFloat64, Temp: math.MaxFloat64, GHz: math.MaxFloat64},
		{Power: math.NaN(), Temp: math.Inf(-1), GHz: math.Inf(1)},
		{Power: -1, Temp: -0.001, GHz: -1},
	}
	boxes := [...]quietBeatBox{{20, 37, 280, 68}, {196, 146, 108, 32}, {196, 207, 108, 27}}
	for _, data := range cases {
		objects := theme.Screen(data)
		if len(objects) != 10 {
			t.Fatalf("got %d objects for %+v", len(objects), data)
		}
		for index, object := range objects {
			if object.Function == nil {
				t.Fatalf("nil object %d for %+v", index, data)
			}
			for _, p := range [...][2]int{{-1, 0}, {0, -1}, {320, 120}, {160, 240}} {
				if object.Function(p[0], p[1]).Exists {
					t.Fatalf("object %d draws outside the screen at %v", index, p)
				}
			}
			if index < 4 {
				continue
			}
			box := boxes[(index-4)/2]
			visible := 0
			for y := 0; y < deepcool.Height; y++ {
				for x := 0; x < deepcool.Width; x++ {
					p := object.Function(x, y)
					if !p.Exists {
						continue
					}
					visible++
					if p.Alpha < 1 || p.Alpha > 255 {
						t.Fatalf("invalid alpha %d in metric object %d", p.Alpha, index)
					}
					if x < box.x || y < box.y || x >= box.x+box.w || y >= box.y+box.h {
						t.Fatalf("metric object %d overflows its box at %d,%d for %+v", index, x, y, data)
					}
				}
			}
			if visible == 0 {
				t.Fatalf("invisible metric object %d for %+v", index, data)
			}
		}
	}
}

func TestQuietBeatFormat(t *testing.T) {
	for _, tc := range []struct {
		value       float64
		decimals    int
		nonnegative bool
		want        string
	}{
		{0, 0, true, "0"},
		{230.4, 0, true, "230"},
		{5.7, 1, true, "5.7"},
		{-40, 0, false, "-40"},
		{-0.001, 0, false, "0"},
		{-1, 1, true, "--"},
		{math.NaN(), 0, false, "--"},
		{math.Inf(1), 1, true, "--"},
		{math.MaxFloat64, 0, true, "--"},
	} {
		if got := quietBeatFormat(tc.value, tc.decimals, tc.nonnegative); got != tc.want {
			t.Errorf("format(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
