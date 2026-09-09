package deepcool

const (
	Width  = 320
	Height = 240
)

type Coords struct {
	X int
	Y int
}

type Frame struct {
	frame []FrameObject
}

type FramePixel struct {
	Exists bool
	Color  Color
	Alpha  int
}

type FrameObject struct {
	Type     string
	Function func(x int, y int) FramePixel
}

func (f *Frame) Add(obj FrameObject) {
	f.frame = append(f.frame, obj)
}

func (f *Frame) AddAll(objs []FrameObject) {
	f.frame = append(f.frame, objs...)
}

func (f *Frame) render() []byte {
	frameBuffer := make([]byte, Width*Height*2)

	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			i := (y*Width + x) * 2

			var out [2]byte

			for _, obj := range f.frame {
				if obj.Function == nil {
					continue
				}
				pixel := obj.Function(x, y)
				if !pixel.Exists {
					continue
				}
				alpha := pixel.Alpha
				if alpha <= 0 {
					alpha = 255
				}
				out = blendRGB565(out, pixel.Color.Bytes(), alpha)
			}

			frameBuffer[i] = out[0]
			frameBuffer[i+1] = out[1]
		}
	}
	return frameBuffer
}

func blendRGB565(bg, fg [2]byte, alpha int) [2]byte {
	if alpha <= 0 {
		return bg
	}

	if alpha >= 255 {
		return fg
	}

	bgValue := uint16(bg[0]) | uint16(bg[1])<<8
	fgValue := uint16(fg[0]) | uint16(fg[1])<<8

	// RGB565 -> RGB888
	br := int((bgValue>>11)&0x1F) * 255 / 31
	bgGreen := int((bgValue>>5)&0x3F) * 255 / 63
	bb := int(bgValue&0x1F) * 255 / 31

	fr := int((fgValue>>11)&0x1F) * 255 / 31
	fgGreen := int((fgValue>>5)&0x3F) * 255 / 63
	fb := int(fgValue&0x1F) * 255 / 31

	// Alpha blend
	r := (fr*alpha + br*(255-alpha)) / 255
	g := (fgGreen*alpha + bgGreen*(255-alpha)) / 255
	b := (fb*alpha + bb*(255-alpha)) / 255

	// RGB888 -> RGB565
	r5 := uint16(r * 31 / 255)
	g6 := uint16(g * 63 / 255)
	b5 := uint16(b * 31 / 255)

	value := (r5 << 11) | (g6 << 5) | b5

	return [2]byte{
		byte(value),
		byte(value >> 8),
	}
}
