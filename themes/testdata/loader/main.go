package main

import (
	"fmt"
	"os"

	"Lm360Go/themes"
)

func main() {
	loader := themes.NewThemeLoaderService()
	theme, err := loader.Load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	objects := theme.Screen(themes.CPUData{CPU: 37.5, Power: 82.25, GHz: 4.2, Temp: 63.5})
	if len(objects) == 0 {
		fmt.Fprintln(os.Stderr, "theme returned no frame objects")
		os.Exit(1)
	}
	fmt.Println(objects[0].Type)
}
