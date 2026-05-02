package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// RenderCanvasToImage converts canvas array to PNG image
func RenderCanvasToImage(canvas [][]int, outputPath string) error {
	height := len(canvas)
	width := len(canvas[0])
	
	img := image.NewGray(image.Rect(0, 0, width, height))
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Canvas value: 255 = white, 0 = black
			// Direct copy
			img.SetGray(x, y, color.Gray{Y: uint8(canvas[y][x])})
		}
	}
	
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return png.Encode(file, img)
}
