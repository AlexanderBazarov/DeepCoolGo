package main

import (
	"fmt"

	deepcool "Lm360Go/deep_cool"
	"Lm360Go/themes"
)

type CustomTheme struct{}

func (c CustomTheme) Screen(data themes.CPUData) []deepcool.FrameObject {
	return []deepcool.FrameObject{
		deepcool.Rectangle(deepcool.Coords{}, deepcool.Coords{X: deepcool.Width - 1, Y: deepcool.Height - 1}, 0, deepcool.ColorBlack),
		deepcool.TextCentered("CUSTOM", deepcool.Coords{X: deepcool.Width / 2, Y: 24}, deepcool.FontSansBold, 20, deepcool.ColorCyan),
		deepcool.TextCentered(fmt.Sprintf("CPU %.0f%%", data.CPU), deepcool.Coords{X: deepcool.Width / 2, Y: 70}, deepcool.FontSansBold, 24, deepcool.ColorWhite),
		deepcool.TextCentered(fmt.Sprintf("POWER %.0f W", data.Power), deepcool.Coords{X: deepcool.Width / 2, Y: 110}, deepcool.FontSansBold, 24, deepcool.ColorWhite),
		deepcool.TextCentered(fmt.Sprintf("FREQ %.1f GHz", data.GHz), deepcool.Coords{X: deepcool.Width / 2, Y: 150}, deepcool.FontSansBold, 24, deepcool.ColorWhite),
		deepcool.TextCentered(fmt.Sprintf("TEMP %.0f °C", data.Temp), deepcool.Coords{X: deepcool.Width / 2, Y: 190}, deepcool.FontSansBold, 24, deepcool.ColorWhite),
	}
}

func Screen(data themes.CPUData) []deepcool.FrameObject {
	return CustomTheme{}.Screen(data)
}

func main() {}
