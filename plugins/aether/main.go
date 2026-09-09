package main

import (
	deepcool "Lm360Go/deep_cool"
	"Lm360Go/themes"
)

func Screen(data themes.CPUData) []deepcool.FrameObject {
	return themes.AetherTheme{}.Screen(data)
}

func main() {}
