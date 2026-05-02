package main

import (
	"image"
	"image/color"
	"math"

	"github.com/disintegration/imaging"
)

// Config holds string art generation parameters
type Config struct {
	NumPins         int
	NumLines        int
	LineWeight      int
	MinDistance     int
	Workers         int
	EdgeWeight      float64 // Edge detection multiplier (prioritize edges)
	Opacity         float64 // v2.1.0: Non-opaque string support (0.0-1.0)
	RandomSampling  bool    // v2.1.0: Random sampling optimization
	SampleSize      int     // v2.1.0: Number of pins to sample per iteration
	AdaptiveStop    bool    // v2.2.0: Adaptive stopping condition
	StopThreshold   float64 // v2.2.0: Quality plateau threshold
	LookAhead       bool    // v2.2.0: Look-ahead optimization (1-step minimax)
}

// Pin represents a point on the circle
type Pin struct {
	X, Y float64
}

// Line represents a connection between two pins
type Line struct {
	From, To int
	Score    float64
}

// LoadImage loads an image from file and converts to grayscale
func LoadImage(path string) (image.Image, error) {
	img, err := imaging.Open(path)
	if err != nil {
		return nil, err
	}

	// Resize to 600x600 for processing
	img = imaging.Resize(img, 600, 600, imaging.Lanczos)
	
	// Convert to grayscale
	gray := imaging.Grayscale(img)
	
	return gray, nil
}

// PreprocessImage applies edge detection (Sobel filter)
// Returns both processed image and edge map
func PreprocessImage(img image.Image) (*image.Gray, *image.Gray) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	// Convert to Gray if not already
	gray, ok := img.(*image.Gray)
	if !ok {
		gray = image.NewGray(bounds)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				gray.Set(x, y, img.At(x, y))
			}
		}
	}

	// Apply Sobel edge detection
	edges := image.NewGray(bounds)
	
	// Sobel kernels
	sobelX := [][]int{
		{-1, 0, 1},
		{-2, 0, 2},
		{-1, 0, 1},
	}
	sobelY := [][]int{
		{-1, -2, -1},
		{0, 0, 0},
		{1, 2, 1},
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var gx, gy float64

			// Apply Sobel kernels
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := float64(gray.GrayAt(x+kx, y+ky).Y)
					gx += pixel * float64(sobelX[ky+1][kx+1])
					gy += pixel * float64(sobelY[ky+1][kx+1])
				}
			}

			// Gradient magnitude
			magnitude := math.Sqrt(gx*gx + gy*gy)
			if magnitude > 255 {
				magnitude = 255
			}

			edges.SetGray(x, y, color.Gray{Y: uint8(magnitude)})
		}
	}

	// Combine original with edges (70% edges, 30% original)
	result := image.NewGray(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeVal := float64(edges.GrayAt(x, y).Y)
			origVal := float64(gray.GrayAt(x, y).Y)
			combined := edgeVal*0.7 + origVal*0.3
			result.SetGray(x, y, color.Gray{Y: uint8(combined)})
		}
	}

	return result, edges
}

// GeneratePins creates evenly spaced pins around a circle
func GeneratePins(numPins int, radius float64, centerX, centerY float64) []Pin {
	pins := make([]Pin, numPins)
	angleStep := 2 * math.Pi / float64(numPins)

	for i := 0; i < numPins; i++ {
		angle := float64(i) * angleStep
		pins[i] = Pin{
			X: centerX + radius*math.Cos(angle),
			Y: centerY + radius*math.Sin(angle),
		}
	}

	return pins
}

// GetPixelsOnLine returns all pixel coordinates on a line using Bresenham's algorithm
func GetPixelsOnLine(x0, y0, x1, y1 int) [][2]int {
	var pixels [][2]int

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	x, y := x0, y0

	for {
		pixels = append(pixels, [2]int{x, y})

		if x == x1 && y == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}

	return pixels
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
