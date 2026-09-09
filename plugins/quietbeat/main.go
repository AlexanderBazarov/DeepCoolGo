package main

import (
	deepcool "Lm360Go/deep_cool"
	"Lm360Go/themes"
)

// Screen is the exact symbol and signature required by ThemeLoaderService.
func Screen(data themes.CPUData) []deepcool.FrameObject {
	return themes.QuietBeatTheme{}.Screen(data)
}

func main() {}
