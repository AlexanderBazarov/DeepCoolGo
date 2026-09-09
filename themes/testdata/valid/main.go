package main

import (
	"fmt"

	deepcool "Lm360Go/deep_cool"
	"Lm360Go/themes"
)

func Screen(data themes.CPUData) []deepcool.FrameObject {
	return []deepcool.FrameObject{
		{Type: fmt.Sprintf("%g/%g/%g/%g", data.CPU, data.Power, data.GHz, data.Temp)},
	}
}
