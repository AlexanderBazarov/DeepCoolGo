package main

import (
	deepcool "Lm360Go/deep_cool"
	"Lm360Go/themes"
)

func Screen(data themes.CPUData) []deepcool.FrameObject {
	theme := themes.StatsTheme{MaxTemp: 100, MaxWatts: 250}
	return theme.Screen(data)
}

func main() {}
