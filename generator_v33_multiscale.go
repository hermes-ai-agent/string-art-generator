package main

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// GenerateStringArtV33MultiScale implements v5 baseline with multi-scale edge detection:
// 1. Combine Sobel + Canny + Laplacian for multi-scale edge map
// 2. Weighted importance map: coarse + fine + texture
// 3. Use proven v5 greedy algorithm for line addition
// 4. Enhanced SSIM-based line removal pass
func GenerateStringArtV33MultiScale(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v33.0 Multi-Scale Edge Detection ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)

	// Create target array
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y)
		}
	}

	// Create canvas (starts white)
	canvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 255.0
		}
	}

	// Generate pins
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := math.Min(centerX, centerY) - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	// Create multi-scale importance map
	importance := createMultiScaleImportanceMap(img, edgeMap, width, height, config.EdgeWeight)

	// Pre-compute line pixels
	fmt.Println("Pre-computing line pixels...")
	linePixels := precomputeSimpleLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Phase 1: Greedy addition with multi-scale importance
	fmt.Println("\n--- Phase 1: Greedy Addition with Multi-Scale Importance ---")

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV33(currentPin, pins, canvas, target, importance,
			linePixels, float64(config.LineWeight), usedLines, config)

		if bestLine.Score <= 0.0001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Draw line on canvas
		drawLineV33(canvas, pins[bestLine.From], pins[bestLine.To], float64(config.LineWeight), width, height)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSEV33(canvas, target, width, height)
			ssim := quickSSIMV33(canvas, target, width, height)
			meanBrightness := calculateMeanBrightnessV33(canvas, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM: %.4f, brightness: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, meanBrightness)
		}
	}

	fmt.Printf("\nPhase 1 complete: %d lines added\n", len(lines))
	initialSSIM := quickSSIMV33(canvas, target, width, height)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	// Phase 2: Enhanced SSIM-based removal
	fmt.Println("\n--- Phase 2: Enhanced SSIM-Based Removal ---")
	lines = enhancedSSIMRemovalV33(lines, pins, canvas, target, width, height, config)

	return lines, canvas
}

// createMultiScaleImportanceMap combines multiple edge detection scales
func createMultiScaleImportanceMap(img *image.Gray, edgeMap *image.Gray, width, height int, edgeWeight float64) [][]float64 {
	importance := make([][]float64, height)
	
	// Compute Laplacian for texture details
	laplacian := computeLaplacian(img, width, height)
	
	// Normalize edge maps
	maxEdge := 0.0
	maxLaplacian := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)
			if edgeVal > maxEdge {
				maxEdge = edgeVal
			}
			if laplacian[y][x] > maxLaplacian {
				maxLaplacian = laplacian[y][x]
			}
		}
	}
	
	// Combine scales: 60% Canny (fine edges) + 40% Laplacian (texture)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			cannyVal := float64(edgeMap.GrayAt(x, y).Y) / maxEdge
			laplacianVal := laplacian[y][x] / maxLaplacian
			
			// Weighted combination
			combined := 0.6*cannyVal + 0.4*laplacianVal
			importance[y][x] = 1.0 + combined*edgeWeight
		}
	}
	
	return importance
}

// computeLaplacian computes Laplacian of Gaussian for texture detection
func computeLaplacian(img *image.Gray, width, height int) [][]float64 {
	laplacian := make([][]float64, height)
	for y := 0; y < height; y++ {
		laplacian[y] = make([]float64, width)
	}
	
	// Laplacian kernel: [0 1 0; 1 -4 1; 0 1 0]
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			center := float64(img.GrayAt(x, y).Y)
			top := float64(img.GrayAt(x, y-1).Y)
			bottom := float64(img.GrayAt(x, y+1).Y)
			left := float64(img.GrayAt(x-1, y).Y)
			right := float64(img.GrayAt(x+1, y).Y)
			
			lap := math.Abs(top + bottom + left + right - 4*center)
			laplacian[y][x] = lap
		}
	}
	
	return laplacian
}

// findBestLineV33 finds the best line from current pin using MSE-based scoring with multi-scale importance
func findBestLineV33(currentPin int, pins []Pin, canvas, target, importance [][]float64,
	linePixels map[[2]int][][2]int, lineWeight float64, usedLines map[[2]int]int, config *Config) Line {

	bestScore := -1e9
	bestTo := -1

	for to := 0; to < len(pins); to++ {
		if to == currentPin {
			continue
		}

		// Check minimum distance
		pinDist := abs(to - currentPin)
		if pinDist < config.MinDistance && pinDist > 0 {
			if len(pins)-pinDist < config.MinDistance {
				continue
			}
		}

		// Get precomputed pixels
		key := [2]int{min(currentPin, to), max(currentPin, to)}
		pixels, exists := linePixels[key]
		if !exists || len(pixels) == 0 {
			continue
		}

		// Calculate weighted error reduction
		score := 0.0
		for _, p := range pixels {
			x, y := p[0], p[1]
			if x >= 0 && x < len(canvas[0]) && y >= 0 && y < len(canvas) {
				currentVal := canvas[y][x]
				targetVal := target[y][x]
				newVal := math.Max(0, currentVal-lineWeight)

				// Error reduction with importance weighting
				currentError := math.Abs(currentVal - targetVal)
				newError := math.Abs(newVal - targetVal)
				errorReduction := (currentError - newError) * importance[y][x]

				score += errorReduction
			}
		}

		if score > bestScore {
			bestScore = score
			bestTo = to
		}
	}

	return Line{From: currentPin, To: bestTo, Score: bestScore}
}

// drawLineV33 draws a line on the canvas
func drawLineV33(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	x0, y0 := int(from.X), int(from.Y)
	x1, y1 := int(to.X), int(to.Y)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy

	for {
		if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
			canvas[y0][x0] = math.Max(0, canvas[y0][x0]-weight)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// enhancedSSIMRemovalV33 performs SSIM-based line removal
func enhancedSSIMRemovalV33(lines []Line, pins []Pin, canvas, target [][]float64,
	width, height int, config *Config) []Line {

	// Sample 15% of lines for removal evaluation
	sampleSize := len(lines) / 7
	if sampleSize < 150 {
		sampleSize = min(150, len(lines))
	}

	// Evaluate each sampled line's SSIM contribution
	type LineContrib struct {
		index        int
		contribution float64
	}
	contribs := make([]LineContrib, 0)

	for i := 0; i < len(lines); i += len(lines) / sampleSize {
		// Remove line temporarily
		tempCanvas := copyCanvasV33(canvas, width, height)
		removeLineV33(tempCanvas, pins[lines[i].From], pins[lines[i].To], float64(config.LineWeight), width, height)

		// Calculate SSIM without this line
		ssimWithout := quickSSIMV33(tempCanvas, target, width, height)
		ssimWith := quickSSIMV33(canvas, target, width, height)

		// Contribution is how much SSIM decreases when line is removed
		contribution := ssimWith - ssimWithout

		contribs = append(contribs, LineContrib{index: i, contribution: contribution})
	}

	// Sort by contribution (ascending)
	sort.Slice(contribs, func(i, j int) bool {
		return contribs[i].contribution < contribs[j].contribution
	})

	// Remove lines with negative contribution (hurt SSIM)
	initialSSIM := quickSSIMV33(canvas, target, width, height)
	removed := 0

	for _, lc := range contribs {
		if lc.contribution > 0 {
			break // Stop when we reach lines with positive contribution
		}

		// Remove line
		removeLineV33(canvas, pins[lines[lc.index].From], pins[lines[lc.index].To],
			float64(config.LineWeight), width, height)

		newSSIM := quickSSIMV33(canvas, target, width, height)
		if newSSIM >= initialSSIM {
			removed++
			lines[lc.index].Score = -1 // Mark for removal
		} else {
			// Restore line if SSIM got worse
			drawLineV33(canvas, pins[lines[lc.index].From], pins[lines[lc.index].To],
				float64(config.LineWeight), width, height)
		}
	}

	// Filter out removed lines
	filteredLines := make([]Line, 0)
	for _, line := range lines {
		if line.Score >= 0 {
			filteredLines = append(filteredLines, line)
		}
	}

	finalSSIM := quickSSIMV33(canvas, target, width, height)
	fmt.Printf("Removed %d lines (SSIM: %.4f -> %.4f)\n", removed, initialSSIM, finalSSIM)

	return filteredLines
}

// Helper functions

func copyCanvasV33(canvas [][]float64, width, height int) [][]float64 {
	temp := make([][]float64, height)
	for y := 0; y < height; y++ {
		temp[y] = make([]float64, width)
		copy(temp[y], canvas[y])
	}
	return temp
}

func removeLineV33(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	x0, y0 := int(from.X), int(from.Y)
	x1, y1 := int(to.X), int(to.Y)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy

	for {
		if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
			canvas[y0][x0] = math.Min(255, canvas[y0][x0]+weight)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func calculateMSEV33(canvas, target [][]float64, width, height int) float64 {
	mse := 0.0
	count := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			mse += diff * diff
			count++
		}
	}
	return mse / float64(count)
}

func quickSSIMV33(canvas, target [][]float64, width, height int) float64 {
	// Quick SSIM approximation using local windows
	windowSize := 11
	c1 := 6.5025  // (0.01 * 255)^2
	c2 := 58.5225 // (0.03 * 255)^2

	ssimSum := 0.0
	count := 0

	for y := windowSize; y < height-windowSize; y += windowSize {
		for x := windowSize; x < width-windowSize; x += windowSize {
			// Calculate local means
			mean1, mean2 := 0.0, 0.0
			for dy := -windowSize / 2; dy <= windowSize/2; dy++ {
				for dx := -windowSize / 2; dx <= windowSize/2; dx++ {
					mean1 += canvas[y+dy][x+dx]
					mean2 += target[y+dy][x+dx]
				}
			}
			n := float64((windowSize + 1) * (windowSize + 1))
			mean1 /= n
			mean2 /= n

			// Calculate local variances and covariance
			var1, var2, covar := 0.0, 0.0, 0.0
			for dy := -windowSize / 2; dy <= windowSize/2; dy++ {
				for dx := -windowSize / 2; dx <= windowSize/2; dx++ {
					diff1 := canvas[y+dy][x+dx] - mean1
					diff2 := target[y+dy][x+dx] - mean2
					var1 += diff1 * diff1
					var2 += diff2 * diff2
					covar += diff1 * diff2
				}
			}
			var1 /= n
			var2 /= n
			covar /= n

			// SSIM formula
			numerator := (2*mean1*mean2 + c1) * (2*covar + c2)
			denominator := (mean1*mean1 + mean2*mean2 + c1) * (var1 + var2 + c2)
			ssim := numerator / denominator

			ssimSum += ssim
			count++
		}
	}

	if count == 0 {
		return 0.0
	}
	return ssimSum / float64(count)
}

func calculateMeanBrightnessV33(canvas [][]float64, width, height int) float64 {
	sum := 0.0
	count := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum += canvas[y][x]
			count++
		}
	}
	return sum / float64(count)
}
