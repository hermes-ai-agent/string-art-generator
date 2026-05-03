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
	MinDistance      int
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

	// Resize to 800x800 for better quality (was 600x600)
	img = imaging.Resize(img, 800, 800, imaging.Lanczos)
	
	// Convert to grayscale
	gray := imaging.Grayscale(img)
	
	return gray, nil
}

// LoadImageRGBA loads an image with alpha channel preserved
func LoadImageRGBA(path string) (*image.NRGBA, error) {
	img, err := imaging.Open(path)
	if err != nil {
		return nil, err
	}

	// Resize to 800x800 for processing
	img = imaging.Resize(img, 800, 800, imaging.Lanczos)
	
	// Convert to NRGBA (preserves alpha)
	nrgba := imaging.Clone(img)
	
	return nrgba, nil
}

// PreprocessImage applies multi-scale edge detection (v36.0)
// Returns the RAW grayscale image and multi-scale edge map
// IMPORTANT: The raw grayscale is the TARGET for string art generation
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

	// Apply slight contrast enhancement to the grayscale
	enhanced := enhanceContrast(gray, width, height)

	// v36.0: Multi-scale edge detection
	// Combine edges at multiple scales to capture both fine details and strong structures
	edges := multiScaleEdgeDetection(enhanced, width, height)

	// Return the ENHANCED grayscale (not edge-blended!) and edge map
	return enhanced, edges
}

// multiScaleEdgeDetection combines Sobel edges at multiple scales (v36.0)
// Scale 1x: Fine details (hair, texture)
// Scale 2x: Medium structures (facial features)
// Scale 4x: Strong structures (face outline, major contours)
func multiScaleEdgeDetection(img *image.Gray, width, height int) *image.Gray {
	bounds := img.Bounds()
	
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

	// Scale 1: Fine details (1x1 sampling)
	edges1x := make([][]float64, height)
	for y := 0; y < height; y++ {
		edges1x[y] = make([]float64, width)
	}
	
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var gx, gy float64
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := float64(img.GrayAt(x+kx, y+ky).Y)
					gx += pixel * float64(sobelX[ky+1][kx+1])
					gy += pixel * float64(sobelY[ky+1][kx+1])
				}
			}
			edges1x[y][x] = math.Sqrt(gx*gx + gy*gy)
		}
	}

	// Scale 2: Medium structures (2x2 sampling)
	edges2x := make([][]float64, height)
	for y := 0; y < height; y++ {
		edges2x[y] = make([]float64, width)
	}
	
	for y := 2; y < height-2; y++ {
		for x := 2; x < width-2; x++ {
			var gx, gy float64
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					// Sample at 2x spacing
					sx, sy := x+kx*2, y+ky*2
					if sx >= 0 && sx < width && sy >= 0 && sy < height {
						pixel := float64(img.GrayAt(sx, sy).Y)
						gx += pixel * float64(sobelX[ky+1][kx+1])
						gy += pixel * float64(sobelY[ky+1][kx+1])
					}
				}
			}
			edges2x[y][x] = math.Sqrt(gx*gx + gy*gy)
		}
	}

	// Scale 3: Strong structures (4x4 sampling)
	edges4x := make([][]float64, height)
	for y := 0; y < height; y++ {
		edges4x[y] = make([]float64, width)
	}
	
	for y := 4; y < height-4; y++ {
		for x := 4; x < width-4; x++ {
			var gx, gy float64
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					// Sample at 4x spacing
					sx, sy := x+kx*4, y+ky*4
					if sx >= 0 && sx < width && sy >= 0 && sy < height {
						pixel := float64(img.GrayAt(sx, sy).Y)
						gx += pixel * float64(sobelX[ky+1][kx+1])
						gy += pixel * float64(sobelY[ky+1][kx+1])
					}
				}
			}
			edges4x[y][x] = math.Sqrt(gx*gx + gy*gy)
		}
	}

	// Combine scales with weighted average
	// Fine details (1x): 40% - captures texture
	// Medium (2x): 35% - captures features
	// Strong (4x): 25% - captures structure
	result := image.NewGray(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			combined := edges1x[y][x]*0.40 + edges2x[y][x]*0.35 + edges4x[y][x]*0.25
			if combined > 255 {
				combined = 255
			}
			result.SetGray(x, y, color.Gray{Y: uint8(combined)})
		}
	}

	return result
}

// enhanceContrast applies contrast enhancement with gamma correction
// For string art with opaque strokes, we need to LIGHTEN the target
// because opaque lines accumulate very quickly to solid black.
// We apply gamma > 1 to lift shadows and prevent over-darkening.
func enhanceContrast(img *image.Gray, width, height int) *image.Gray {
	// Calculate histogram
	var histogram [256]int
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			histogram[img.GrayAt(x, y).Y]++
		}
	}

	// Find 2% and 98% percentiles for contrast stretching
	totalPixels := width * height
	lowThreshold := int(float64(totalPixels) * 0.02)
	highThreshold := int(float64(totalPixels) * 0.98)

	cumSum := 0
	lowVal := 0
	highVal := 255

	for i := 0; i < 256; i++ {
		cumSum += histogram[i]
		if cumSum >= lowThreshold && lowVal == 0 {
			lowVal = i
		}
		if cumSum >= highThreshold {
			highVal = i
			break
		}
	}

	// Apply contrast stretching + gamma 1.8 correction
	// Gamma 1.8 lifts shadows into the achievable range for string art.
	// String art physically cannot reach very dark values (mean ~140-150 at best),
	// so we remap the target to a range that string art CAN achieve.
	// This produces perceptually better results because the optimizer can
	// actually match the target instead of always being "too bright".
	gamma := 1.0 // No gamma - keep target dark for maximum contrast
	
	result := image.NewGray(img.Bounds())
	rangeVal := float64(highVal - lowVal)
	if rangeVal < 1 {
		rangeVal = 1
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			val := float64(img.GrayAt(x, y).Y)
			// Stretch to full range
			stretched := (val - float64(lowVal)) / rangeVal
			if stretched < 0 {
				stretched = 0
			}
			if stretched > 1 {
				stretched = 1
			}
			// Apply gamma (lifts shadows)
			gammaApplied := math.Pow(stretched, 1.0/gamma) * 255.0
			result.SetGray(x, y, color.Gray{Y: uint8(gammaApplied)})
		}
	}

	return result
}

// PreprocessImageRGBA applies edge detection with alpha channel awareness
func PreprocessImageRGBA(img *image.NRGBA) (*image.NRGBA, *image.Gray) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	// Convert to grayscale for edge detection
	gray := image.NewGray(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Standard grayscale conversion
			grayVal := uint8((r*299 + g*587 + b*114) / 1000 / 256)
			gray.SetGray(x, y, color.Gray{Y: grayVal})
		}
	}
	
	// Apply Sobel edge detection
	edges := image.NewGray(bounds)
	
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

			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := float64(gray.GrayAt(x+kx, y+ky).Y)
					gx += pixel * float64(sobelX[ky+1][kx+1])
					gy += pixel * float64(sobelY[ky+1][kx+1])
				}
			}

			magnitude := math.Sqrt(gx*gx + gy*gy)
			if magnitude > 255 {
				magnitude = 255
			}

			edges.SetGray(x, y, color.Gray{Y: uint8(magnitude)})
		}
	}
	
	// Create result with grayscale + alpha preserved
	result := image.NewNRGBA(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grayVal := gray.GrayAt(x, y).Y
			_, _, _, alpha := img.At(x, y).RGBA()
			result.SetNRGBA(x, y, color.NRGBA{
				R: grayVal,
				G: grayVal,
				B: grayVal,
				A: uint8(alpha >> 8),
			})
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
