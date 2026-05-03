package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// RenderCanvasToImageFloat converts float64 canvas array to PNG image
func RenderCanvasToImageFloat(canvas [][]float64, outputPath string) error {
	height := len(canvas)
	width := len(canvas[0])

	img := image.NewGray(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			val := canvas[y][x]
			if val < 0 {
				val = 0
			}
			if val > 255 {
				val = 255
			}
			img.SetGray(x, y, color.Gray{Y: uint8(val)})
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

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
