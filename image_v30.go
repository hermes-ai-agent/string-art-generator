package main

import (
	"image"
	"image/color"
	"math"
)

// PreprocessImageV30 applies Canny edge detection + morphological operations
// Returns the RAW grayscale image and enhanced edge map
func PreprocessImageV30(img image.Image) (*image.Gray, *image.Gray) {
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

	// Apply contrast enhancement to the grayscale
	enhanced := enhanceContrastV30(gray, width, height)

	// Step 1: Gaussian blur for noise reduction (5x5 kernel)
	blurred := gaussianBlur(enhanced, width, height)

	// Step 2: Compute gradients using Sobel operator
	gradientMag, gradientDir := computeGradients(blurred, width, height)

	// Step 3: Non-maximum suppression
	suppressed := nonMaximumSuppression(gradientMag, gradientDir, width, height)

	// Step 4: Double threshold (high=100, low=50)
	edges := doubleThreshold(suppressed, width, height, 100, 50)

	// Step 5: Edge tracking by hysteresis
	tracked := edgeTracking(edges, width, height)

	// Step 6: Morphological closing (dilation → erosion) to connect broken edges
	dilated := morphologicalDilation(tracked, width, height)
	closed := morphologicalErosion(dilated, width, height)

	// Return the ENHANCED grayscale and Canny edge map
	return enhanced, closed
}

// enhanceContrastV30 applies contrast enhancement with gamma correction
func enhanceContrastV30(img *image.Gray, width, height int) *image.Gray {
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

	// Apply contrast stretching + mild gamma correction
	gamma := 1.2 // Mild shadow lift only
	
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

// gaussianBlur applies 5x5 Gaussian blur for noise reduction
func gaussianBlur(img *image.Gray, width, height int) *image.Gray {
	// 5x5 Gaussian kernel (sigma ≈ 1.4)
	kernel := [][]float64{
		{2, 4, 5, 4, 2},
		{4, 9, 12, 9, 4},
		{5, 12, 15, 12, 5},
		{4, 9, 12, 9, 4},
		{2, 4, 5, 4, 2},
	}
	kernelSum := 159.0 // Sum of all kernel values

	result := image.NewGray(img.Bounds())
	
	for y := 2; y < height-2; y++ {
		for x := 2; x < width-2; x++ {
			var sum float64
			for ky := -2; ky <= 2; ky++ {
				for kx := -2; kx <= 2; kx++ {
					pixel := float64(img.GrayAt(x+kx, y+ky).Y)
					sum += pixel * kernel[ky+2][kx+2]
				}
			}
			result.SetGray(x, y, color.Gray{Y: uint8(sum / kernelSum)})
		}
	}

	return result
}

// computeGradients calculates gradient magnitude and direction using Sobel
func computeGradients(img *image.Gray, width, height int) (*image.Gray, [][]float64) {
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

	magnitude := image.NewGray(img.Bounds())
	direction := make([][]float64, height)
	for i := range direction {
		direction[i] = make([]float64, width)
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var gx, gy float64

			// Apply Sobel kernels
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := float64(img.GrayAt(x+kx, y+ky).Y)
					gx += pixel * float64(sobelX[ky+1][kx+1])
					gy += pixel * float64(sobelY[ky+1][kx+1])
				}
			}

			// Gradient magnitude
			mag := math.Sqrt(gx*gx + gy*gy)
			if mag > 255 {
				mag = 255
			}
			magnitude.SetGray(x, y, color.Gray{Y: uint8(mag)})

			// Gradient direction (in radians)
			direction[y][x] = math.Atan2(gy, gx)
		}
	}

	return magnitude, direction
}

// nonMaximumSuppression thins edges to single-pixel width
func nonMaximumSuppression(magnitude *image.Gray, direction [][]float64, width, height int) *image.Gray {
	result := image.NewGray(magnitude.Bounds())

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			angle := direction[y][x] * 180.0 / math.Pi
			if angle < 0 {
				angle += 180
			}

			mag := float64(magnitude.GrayAt(x, y).Y)
			var neighbor1, neighbor2 float64

			// Quantize angle to 4 directions (0°, 45°, 90°, 135°)
			if (angle >= 0 && angle < 22.5) || (angle >= 157.5 && angle <= 180) {
				// Horizontal edge (0°)
				neighbor1 = float64(magnitude.GrayAt(x+1, y).Y)
				neighbor2 = float64(magnitude.GrayAt(x-1, y).Y)
			} else if angle >= 22.5 && angle < 67.5 {
				// Diagonal edge (45°)
				neighbor1 = float64(magnitude.GrayAt(x+1, y-1).Y)
				neighbor2 = float64(magnitude.GrayAt(x-1, y+1).Y)
			} else if angle >= 67.5 && angle < 112.5 {
				// Vertical edge (90°)
				neighbor1 = float64(magnitude.GrayAt(x, y+1).Y)
				neighbor2 = float64(magnitude.GrayAt(x, y-1).Y)
			} else {
				// Diagonal edge (135°)
				neighbor1 = float64(magnitude.GrayAt(x-1, y-1).Y)
				neighbor2 = float64(magnitude.GrayAt(x+1, y+1).Y)
			}

			// Suppress if not local maximum
			if mag >= neighbor1 && mag >= neighbor2 {
				result.SetGray(x, y, color.Gray{Y: uint8(mag)})
			}
		}
	}

	return result
}

// doubleThreshold applies high and low thresholds to classify edges
func doubleThreshold(img *image.Gray, width, height int, highThreshold, lowThreshold uint8) *image.Gray {
	result := image.NewGray(img.Bounds())

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			val := img.GrayAt(x, y).Y
			if val >= highThreshold {
				// Strong edge
				result.SetGray(x, y, color.Gray{Y: 255})
			} else if val >= lowThreshold {
				// Weak edge
				result.SetGray(x, y, color.Gray{Y: 128})
			}
			// else: non-edge (0)
		}
	}

	return result
}

// edgeTracking connects weak edges to strong edges (hysteresis)
func edgeTracking(img *image.Gray, width, height int) *image.Gray {
	result := image.NewGray(img.Bounds())
	
	// Copy strong edges
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if img.GrayAt(x, y).Y == 255 {
				result.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}

	// Connect weak edges to strong edges
	changed := true
	for changed {
		changed = false
		for y := 1; y < height-1; y++ {
			for x := 1; x < width-1; x++ {
				if img.GrayAt(x, y).Y == 128 && result.GrayAt(x, y).Y == 0 {
					// Check if connected to strong edge
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							if result.GrayAt(x+dx, y+dy).Y == 255 {
								result.SetGray(x, y, color.Gray{Y: 255})
								changed = true
								break
							}
						}
						if changed {
							break
						}
					}
				}
			}
		}
	}

	return result
}

// morphologicalDilation expands edges (3x3 structuring element)
func morphologicalDilation(img *image.Gray, width, height int) *image.Gray {
	result := image.NewGray(img.Bounds())

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			maxVal := uint8(0)
			// Check 3x3 neighborhood
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					val := img.GrayAt(x+dx, y+dy).Y
					if val > maxVal {
						maxVal = val
					}
				}
			}
			result.SetGray(x, y, color.Gray{Y: maxVal})
		}
	}

	return result
}

// morphologicalErosion shrinks edges (3x3 structuring element)
func morphologicalErosion(img *image.Gray, width, height int) *image.Gray {
	result := image.NewGray(img.Bounds())

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			minVal := uint8(255)
			// Check 3x3 neighborhood
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					val := img.GrayAt(x+dx, y+dy).Y
					if val < minVal {
						minVal = val
					}
				}
			}
			result.SetGray(x, y, color.Gray{Y: minVal})
		}
	}

	return result
}
