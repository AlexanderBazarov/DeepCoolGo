package themes

import (
	"fmt"
	"os"
	"plugin"

	deepcool "Lm360Go/deep_cool"
)

type ThemeLoaderService struct{}

func NewThemeLoaderService() *ThemeLoaderService {
	return &ThemeLoaderService{}
}

func (s *ThemeLoaderService) Load(path string) (Theme, error) {
	if path == "" {
		return nil, fmt.Errorf("theme plugin path is empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read theme plugin %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("theme plugin %q is not a regular file", path)
	}

	loaded, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("load theme plugin %q: %w", path, err)
	}
	symbol, err := loaded.Lookup("Screen")
	if err != nil {
		return nil, fmt.Errorf("find Screen in theme plugin %q: %w", path, err)
	}
	screen, ok := symbol.(func(CPUData) []deepcool.FrameObject)
	if !ok {
		return nil, fmt.Errorf("theme plugin %q: Screen must be func(themes.CPUData) []deepcool.FrameObject", path)
	}
	return pluginTheme{screen: screen}, nil
}

type pluginTheme struct {
	screen func(CPUData) []deepcool.FrameObject
}

func (p pluginTheme) Screen(data CPUData) []deepcool.FrameObject {
	return p.screen(data)
}
