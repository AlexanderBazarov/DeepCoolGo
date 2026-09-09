package themes

import deepcool "Lm360Go/deep_cool"

type Theme interface {
	Screen(data CPUData) []deepcool.FrameObject
}
