package main

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// GenerateStringArtV30CannyEdge implements v5 baseline with Canny edge detection:
// 1. Use Canny edge detection + morphological operations for better edge map
// 2. Use proven v5 greedy algorithm for line addition
// 3. Enhanced SSIM-based line removal pass
// 4. Multi-pass removal: remove, re-evaluate, remove again
func GenerateStringArtV30CannyEdge(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v30.0 Canny Edge Detection ===\n")
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

	// Create importance map (edge-weighted with Canny edges)
	importance := createImportanceMapV30(edgeMap, width, height, config.EdgeWeight)

	// Pre-compute line pixels
	fmt.Println("Pre-computing line pixels...")
	linePixels := precomputeSimpleLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Phase 1: Greedy addition (proven v5 algorithm)
	fmt.Println("\n--- Phase 1: Greedy Addition (V5 Algorithm) ---")

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV30(currentPin, pins, canvas, target, importance,
			linePixels, float64(config.LineWeight), usedLines, config)

		if bestLine.Score <= 0.0001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Draw line on canvas
		drawLineV30(canvas, pins[bestLine.From], pins[bestLine.To], float64(config.LineWeight), width, height)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSEV30(canvas, target, width, height)
			ssim := quickSSIMV30(canvas, target, width, height)
			meanBrightness := calculateMeanBrightnessV30(canvas, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM: %.4f, brightness: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, meanBrightness)
		}
	}

	fmt.Printf("\nPhase 1 complete: %d lines added\n", len(lines))
	initialSSIM := quickSSIMV30(canvas, target, width, height)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	// Phase 2: SSIM-based multi-pass removal
	fmt.Println("\n--- Phase 2: SSIM-Based Multi-Pass Removal ---")
	lines = ssimBasedRemovalV30(lines, pins, canvas, target, importance, width, height, config)

	return lines, canvas
}

// createImportanceMapV30 creates edge-weighted importance map with Canny edges
func createImportanceMapV30(edgeMap *image.Gray, width, height int, edgeWeight float64) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			// Canny edges are binary (0 or 255), so we boost them significantly
			importance[y][x] = 1.0 + edgeStrength*edgeWeight
		}
	}
	return importance
}

// findBestLineV30 finds the best line from current pin (greedy MSE-based)
func findBestLineV30(currentPin int, pins []Pin, canvas [][]float64, target [][]float64,
	importance [][]float64, linePixels map[[2]int][][2]int, lineWeight float64,
	usedLines map[[2]int]int, config *Config) Line {

	bestScore := -1.0
	bestTo := -1

	for to := 0; to < len(pins); to++ {
		if to == currentPin {
			continue
		}

		// Check minimum distance
		dist := abs(to - currentPin)
		if dist > len(pins)/2 {
			dist = len(pins) - dist
		}
		if dist < config.MinDistance {
			continue
		}

		// Get pre-computed pixels
		key := [2]int{min(currentPin, to), max(currentPin, to)}
		pixels, exists := linePixels[key]
		if !exists {
			continue
		}

		// Calculate improvement score (MSE reduction)
		score := 0.0
		for _, pixel := range pixels {
			x, y := pixel[0], pixel[1]
			if x < 0 || x >= len(canvas[0]) || y < 0 || y >= len(canvas) {
				continue
			}

			currentVal := canvas[y][x]
			targetVal := target[y][x]
			newVal := currentVal - lineWeight
			if newVal < 0 {
				newVal = 0
			}

			// MSE improvement with importance weighting
			currentError := (currentVal - targetVal) * (currentVal - targetVal)
			newError := (newVal - targetVal) * (newVal - targetVal)
			improvement := (currentError - newError) * importance[y][x]

			score += improvement
		}

		if score > bestScore {
			bestScore = score
			bestTo = to
		}
	}

	return Line{From: currentPin, To: bestTo, Score: bestScore}
}

// drawLineV30 draws a line on the canvas
func drawLineV30(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	pixels := GetPixelsOnLine(int(from.X), int(from.Y), int(to.X), int(to.Y))
	for _, pixel := range pixels {
		x, y := pixel[0], pixel[1]
		if x >= 0 && x < width && y >= 0 && y < height {
			canvas[y][x] -= weight
			if canvas[y][x] < 0 {
				canvas[y][x] = 0
			}
		}
	}
}

// undrawLineV30 removes a line from the canvas
func undrawLineV30(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	pixels := GetPixelsOnLine(int(from.X), int(from.Y), int(to.X), int(to.Y))
	for _, pixel := range pixels {
		x, y := pixel[0], pixel[1]
		if x >= 0 && x < width && y >= 0 && y < height {
			canvas[y][x] += weight
			if canvas[y][x] > 255 {
				canvas[y][x] = 255
			}
		}
	}
}

// ssimBasedRemovalV30 performs intelligent line removal using SSIM scoring
func ssimBasedRemovalV30(lines []Line, pins []Pin, canvas [][]float64, target [][]float64,
	importance [][]float64, width, height int, config *Config) []Line {

	if len(lines) == 0 {
		return lines
	}

	// Pass 1: Calculate SSIM contribution for each line
	fmt.Println("\nPass 1: Calculating SSIM contributions...")

	type removalCandidate struct {
		index           int
		ssimImprovement float64
	}

	candidates := make([]removalCandidate, 0)

	// Sample every 10th line for faster evaluation
	sampleInterval := 10
	if len(lines) < 1000 {
		sampleInterval = 5
	}

	for i := 0; i < len(lines); i += sampleInterval {
		// Temporarily remove line
		line := lines[i]
		undrawLineV30(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)

		// Calculate SSIM without this line
		ssimWithout := quickSSIMV30(canvas, target, width, height)

		// Restore line
		drawLineV30(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)

		// Calculate SSIM with this line
		ssimWith := quickSSIMV30(canvas, target, width, height)

		// If removing improves SSIM, it's a candidate
		improvement := ssimWithout - ssimWith

		if improvement > 0.0001 {
			candidates = append(candidates, removalCandidate{
				index:           i,
				ssimImprovement: improvement,
			})
		}
	}

	fmt.Printf("Found %d removal candidates (sampled %d lines)\n", len(candidates), len(lines)/sampleInterval)

	if len(candidates) == 0 {
		fmt.Println("No lines to remove (all contribute positively)")
		return lines
	}

	// Sort by SSIM improvement (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ssimImprovement > candidates[j].ssimImprovement
	})

	// Pass 2: Remove top candidates
	maxRemoval := len(lines) / 20 // Remove up to 5% of lines
	if maxRemoval < 10 {
		maxRemoval = 10
	}
	if maxRemoval > 200 {
		maxRemoval = 200
	}

	fmt.Printf("\nPass 2: Removing up to %d lines...\n", maxRemoval)

	removedIndices := make(map[int]bool)
	removedCount := 0
	initialSSIM := quickSSIMV30(canvas, target, width, height)

	for _, candidate := range candidates {
		if removedCount >= maxRemoval {
			break
		}

		idx := candidate.index
		line := lines[idx]

		// Remove line
		undrawLineV30(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)

		// Check if SSIM improved
		newSSIM := quickSSIMV30(canvas, target, width, height)

		if newSSIM > initialSSIM {
			// Keep removed
			removedIndices[idx] = true
			removedCount++
			initialSSIM = newSSIM
			if removedCount%20 == 0 {
				fmt.Printf("Removed %d lines, SSIM: %.4f (+%.4f)\n",
					removedCount, newSSIM, newSSIM-quickSSIMV30(canvas, target, width, height))
			}
		} else {
			// Restore line
			drawLineV30(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)
		}
	}

	// Build final line list
	finalLines := make([]Line, 0, len(lines)-removedCount)
	for i, line := range lines {
		if !removedIndices[i] {
			finalLines = append(finalLines, line)
		}
	}

	fmt.Printf("\nRemoval complete: %d lines removed, %d lines remaining\n", removedCount, len(finalLines))
	finalSSIM := quickSSIMV30(canvas, target, width, height)
	fmt.Printf("Final SSIM: %.4f (improvement: +%.4f)\n", finalSSIM, finalSSIM-initialSSIM)

	return finalLines
}

// calculateMSEV30 calculates mean squared error
func calculateMSEV30(canvas [][]float64, target [][]float64, width, height int) float64 {
	sum := 0.0
	count := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			sum += diff * diff
			count++
		}
	}
	return sum / float64(count)
}

// quickSSIMV30 calculates a fast approximation of SSIM
func quickSSIMV30(canvas [][]float64, target [][]float64, width, height int) float64 {
	// Constants for SSIM
	c1 := 6.5025  // (0.01 * 255)^2
	c2 := 58.5225 // (0.03 * 255)^2

	windowSize := 8
	stride := 4

	ssimSum := 0.0
	windowCount := 0

	for y := 0; y <= height-windowSize; y += stride {
		for x := 0; x <= width-windowSize; x += stride {
			// Calculate means
			var meanCanvas, meanTarget float64
			for wy := 0; wy < windowSize; wy++ {
				for wx := 0; wx < windowSize; wx++ {
					meanCanvas += canvas[y+wy][x+wx]
					meanTarget += target[y+wy][x+wx]
				}
			}
			meanCanvas /= float64(windowSize * windowSize)
			meanTarget /= float64(windowSize * windowSize)

			// Calculate variances and covariance
			var varCanvas, varTarget, covar float64
			for wy := 0; wy < windowSize; wy++ {
				for wx := 0; wx < windowSize; wx++ {
					diffCanvas := canvas[y+wy][x+wx] - meanCanvas
					diffTarget := target[y+wy][x+wx] - meanTarget
					varCanvas += diffCanvas * diffCanvas
					varTarget += diffTarget * diffTarget
					covar += diffCanvas * diffTarget
				}
			}
			varCanvas /= float64(windowSize*windowSize - 1)
			varTarget /= float64(windowSize*windowSize - 1)
			covar /= float64(windowSize*windowSize - 1)

			// SSIM formula
			numerator := (2*meanCanvas*meanTarget + c1) * (2*covar + c2)
			denominator := (meanCanvas*meanCanvas + meanTarget*meanTarget + c1) *
				(varCanvas + varTarget + c2)

			if denominator > 0 {
				ssimSum += numerator / denominator
				windowCount++
			}
		}
	}

	if windowCount == 0 {
		return 0
	}
	return ssimSum / float64(windowCount)
}

// calculateMeanBrightnessV30 calculates mean brightness of canvas
func calculateMeanBrightnessV30(canvas [][]float64, width, height int) float64 {
	sum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum += canvas[y][x]
		}
	}
	return sum / float64(width*height)
}
