package deepcool

type Color uint16

const (
	ColorBlack   Color = 0x0000
	ColorWhite   Color = 0xFFFF
	ColorRed     Color = 0xF800
	ColorGreen   Color = 0x07E0
	ColorBlue    Color = 0x001F
	ColorYellow  Color = 0xFFE0
	ColorCyan    Color = 0x07FF
	ColorMagenta Color = 0xF81F
	ColorOrange  Color = 0xFD20
	ColorGray    Color = 0x8410
)

func RGB(r, g, b uint8) Color {
	return Color(uint16(r&0xF8)<<8 | uint16(g&0xFC)<<3 | uint16(b)>>3)
}

func (c Color) Bytes() [2]byte {
	return [2]byte{byte(c & 0xFF), byte(c >> 8)}
}
