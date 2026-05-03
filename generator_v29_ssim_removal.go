package main

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// GenerateStringArtV29SSIMRemoval implements v5 baseline with SSIM-based line removal:
// 1. Use proven v5 greedy algorithm for line addition
// 2. Enhanced SSIM-based line removal pass (instead of simple MSE-based)
// 3. Multi-pass removal: remove, re-evaluate, remove again
// 4. Contribution tracking for smarter removal decisions
func GenerateStringArtV29SSIMRemoval(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v29.0 SSIM-Based Removal ===\n")
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

	// Create importance map (edge-weighted)
	importance := createImportanceMapV29(edgeMap, width, height, config.EdgeWeight)

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
		bestLine := findBestLineV29(currentPin, pins, canvas, target, importance,
			linePixels, float64(config.LineWeight), usedLines, config)

		if bestLine.Score <= 0.0001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Draw line on canvas
		drawLineV29(canvas, pins[bestLine.From], pins[bestLine.To], float64(config.LineWeight), width, height)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSEV29(canvas, target, width, height)
			ssim := quickSSIMV29(canvas, target, width, height)
			meanBrightness := calculateMeanBrightnessV29(canvas, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM: %.4f, brightness: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, meanBrightness)
		}
	}

	fmt.Printf("\nPhase 1 complete: %d lines added\n", len(lines))
	initialSSIM := quickSSIMV29(canvas, target, width, height)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	// Phase 2: SSIM-based multi-pass removal
	fmt.Println("\n--- Phase 2: SSIM-Based Multi-Pass Removal ---")
	lines = ssimBasedRemovalV29(lines, pins, canvas, target, importance, width, height, config)

	return lines, canvas
}

// ssimBasedRemovalV29 performs intelligent line removal using SSIM scoring
func ssimBasedRemovalV29(lines []Line, pins []Pin, canvas [][]float64, target [][]float64,
	importance [][]float64, width, height int, config *Config) []Line {

	if len(lines) == 0 {
		return lines
	}

	// Track line contributions
	lineContributions := make([]float64, len(lines))

	// Pass 1: Calculate SSIM contribution for each line
	fmt.Println("\nPass 1: Calculating SSIM contributions...")

	type removalCandidate struct {
		index        int
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
		undrawLineV29(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)

		// Calculate SSIM without this line
		ssimWithout := quickSSIMV29(canvas, target, width, height)

		// Restore line
		drawLineV29(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)

		// Calculate SSIM with this line
		ssimWith := quickSSIMV29(canvas, target, width, height)

		// If removing improves SSIM, it's a candidate
		improvement := ssimWithout - ssimWith
		lineContributions[i] = improvement

		if improvement > 0.0001 {
			candidates = append(candidates, removalCandidate{
				index:        i,
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
	initialSSIM := quickSSIMV29(canvas, target, width, height)

	for _, candidate := range candidates {
		if removedCount >= maxRemoval {
			break
		}

		idx := candidate.index
		line := lines[idx]

		// Remove line
		undrawLineV29(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)

		// Check if SSIM improved
		newSSIM := quickSSIMV29(canvas, target, width, height)

		if newSSIM > initialSSIM {
			// Keep removed
			removedIndices[idx] = true
			removedCount++
			initialSSIM = newSSIM
			if removedCount%20 == 0 {
				fmt.Printf("Removed %d lines, SSIM: %.4f (+%.4f)\n",
					removedCount, newSSIM, newSSIM-initialSSIM)
			}
		} else {
			// Restore line (removal didn't help)
			drawLineV29(canvas, pins[line.From], pins[line.To], float64(config.LineWeight), width, height)
		}
	}

	// Build new line list without removed lines
	newLines := make([]Line, 0, len(lines)-removedCount)
	for i, line := range lines {
		if !removedIndices[i] {
			newLines = append(newLines, line)
		}
	}

	finalSSIM := quickSSIMV29(canvas, target, width, height)
	fmt.Printf("\nRemoval complete: removed %d lines (%.1f%%)\n", removedCount, float64(removedCount)/float64(len(lines))*100)
	fmt.Printf("Final SSIM: %.4f (improvement: +%.4f)\n", finalSSIM, finalSSIM-initialSSIM)

	return newLines
}

// undrawLineV29 removes a line from the canvas (inverse of drawLine)
func undrawLineV29(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	x0, y0 := int(from.X), int(from.Y)
	x1, y1 := int(to.X), int(to.Y)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
			// Add back the weight (undo darkening)
			canvas[y0][x0] += weight
			if canvas[y0][x0] > 255 {
				canvas[y0][x0] = 255
			}
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

// quickSSIMV29 calculates SSIM between canvas and target using sliding window
func quickSSIMV29(canvas, target [][]float64, width, height int) float64 {
	windowSize := 11
	c1 := 6.5025  // (0.01 * 255)^2
	c2 := 58.5225 // (0.03 * 255)^2

	ssimSum := 0.0
	count := 0

	for y := 0; y <= height-windowSize; y += windowSize / 2 {
		for x := 0; x <= width-windowSize; x += windowSize / 2 {
			// Calculate means
			meanCanvas := 0.0
			meanTarget := 0.0
			for dy := 0; dy < windowSize; dy++ {
				for dx := 0; dx < windowSize; dx++ {
					if y+dy < height && x+dx < width {
						meanCanvas += canvas[y+dy][x+dx]
						meanTarget += target[y+dy][x+dx]
					}
				}
			}
			meanCanvas /= float64(windowSize * windowSize)
			meanTarget /= float64(windowSize * windowSize)

			// Calculate variances and covariance
			varCanvas := 0.0
			varTarget := 0.0
			covar := 0.0
			for dy := 0; dy < windowSize; dy++ {
				for dx := 0; dx < windowSize; dx++ {
					if y+dy < height && x+dx < width {
						diffCanvas := canvas[y+dy][x+dx] - meanCanvas
						diffTarget := target[y+dy][x+dx] - meanTarget
						varCanvas += diffCanvas * diffCanvas
						varTarget += diffTarget * diffTarget
						covar += diffCanvas * diffTarget
					}
				}
			}
			varCanvas /= float64(windowSize * windowSize)
			varTarget /= float64(windowSize * windowSize)
			covar /= float64(windowSize * windowSize)

			// Calculate SSIM for this window
			numerator := (2*meanCanvas*meanTarget + c1) * (2*covar + c2)
			denominator := (meanCanvas*meanCanvas + meanTarget*meanTarget + c1) * (varCanvas + varTarget + c2)

			if denominator > 0 {
				ssimSum += numerator / denominator
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	return ssimSum / float64(count)
}

// createImportanceMapV29 creates importance map from edge detection
func createImportanceMapV29(edgeMap *image.Gray, width, height int, edgeWeight float64) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			edgeVal := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + edgeVal*edgeWeight
		}
	}
	return importance
}

func findBestLineV29(currentPin int, pins []Pin, canvas, target, importance [][]float64,
	linePixels map[[2]int][][2]int, weight float64, usedLines map[[2]int]int, config *Config) Line {

	type candidate struct {
		to    int
		score float64
	}

	candidates := make([]candidate, 0, config.NumPins)

	// Collect all valid candidates
	for nextPin := 0; nextPin < config.NumPins; nextPin++ {
		if nextPin == currentPin {
			continue
		}

		key := [2]int{min(currentPin, nextPin), max(currentPin, nextPin)}
		if usedLines[key] > 0 {
			continue
		}

		pixels, exists := linePixels[key]
		if !exists {
			continue
		}

		// Calculate score
		score := 0.0
		for _, pixel := range pixels {
			x, y := pixel[0], pixel[1]
			if x >= 0 && x < len(canvas[0]) && y >= 0 && y < len(canvas) {
				diff := canvas[y][x] - target[y][x]
				if diff > 0 {
					score += diff * importance[y][x]
				}
			}
		}

		if score > 0 {
			candidates = append(candidates, candidate{to: nextPin, score: score})
		}
	}

	if len(candidates) == 0 {
		return Line{From: currentPin, To: currentPin, Score: 0}
	}

	// Find best candidate
	bestIdx := 0
	bestScore := candidates[0].score
	for i := 1; i < len(candidates); i++ {
		if candidates[i].score > bestScore {
			bestScore = candidates[i].score
			bestIdx = i
		}
	}

	return Line{
		From:  currentPin,
		To:    candidates[bestIdx].to,
		Score: bestScore,
	}
}

func drawLineV29(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	x0, y0 := int(from.X), int(from.Y)
	x1, y1 := int(to.X), int(to.Y)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
			canvas[y0][x0] -= weight
			if canvas[y0][x0] < 0 {
				canvas[y0][x0] = 0
			}
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

func calculateMSEV29(canvas, target [][]float64, width, height int) float64 {
	mse := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			mse += diff * diff
		}
	}
	return mse / float64(width*height)
}

func calculateMeanBrightnessV29(canvas [][]float64, width, height int) float64 {
	sum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum += canvas[y][x]
		}
	}
	return sum / float64(width*height)
}

// precomputeSimpleLinePixels pre-computes all valid line segments with simple pixel coordinates
func precomputeSimpleLinePixels(pins []Pin, width, height, minDistance int) map[[2]int][][2]int {
	numPins := len(pins)
	result := make(map[[2]int][][2]int)

	for i := 0; i < numPins; i++ {
		for j := i + 1; j < numPins; j++ {
			// Check distance constraint
			distance := j - i
			if distance < minDistance && numPins-distance < minDistance {
				continue
			}

			// Compute line pixels using Bresenham
			pixels := bresenhamLineV29(int(pins[i].X), int(pins[i].Y), int(pins[j].X), int(pins[j].Y))
			
			key := [2]int{i, j}
			result[key] = pixels
		}
	}

	return result
}

// bresenhamLineV29 computes pixels along a line using Bresenham's algorithm
func bresenhamLineV29(x0, y0, x1, y1 int) [][2]int {
	pixels := make([][2]int, 0)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		pixels = append(pixels, [2]int{x0, y0})

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

	return pixels
}
