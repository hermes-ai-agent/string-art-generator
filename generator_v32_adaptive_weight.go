package main

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// GenerateStringArtV32AdaptiveWeight implements v5 baseline with adaptive line weight scheduling:
// 1. Start with heavy lines (weight 35) for main structure
// 2. Gradually decrease to light lines (weight 20) for details
// 3. Use proven v5 greedy algorithm for line addition
// 4. Simple heuristic removal pass (remove lines with negative contribution)
func GenerateStringArtV32AdaptiveWeight(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v32.0 Adaptive Line Weight Scheduling ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d (base), Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)
	fmt.Printf("Adaptive weight: 35 -> 20 (heavy to light)\n")

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
	importance := createImportanceMapV32(edgeMap, width, height, config.EdgeWeight)

	// Pre-compute line pixels
	fmt.Println("Pre-computing line pixels...")
	linePixels := precomputeSimpleLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Phase 1: Greedy addition with adaptive weight scheduling
	fmt.Println("\n--- Phase 1: Greedy Addition with Adaptive Weight Scheduling ---")

	// Weight schedule: 32 -> 22 over numLines iterations
	startWeight := 32.0
	endWeight := 22.0

	for i := 0; i < config.NumLines; i++ {
		// Calculate current weight (linear interpolation)
		progress := float64(i) / float64(config.NumLines)
		currentWeight := startWeight + (endWeight-startWeight)*progress

		bestLine := findBestLineV32(currentPin, pins, canvas, target, importance,
			linePixels, currentWeight, usedLines, config)

		if bestLine.Score <= 0.0001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Draw line on canvas with current weight
		drawLineV32(canvas, pins[bestLine.From], pins[bestLine.To], currentWeight, width, height)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		// Store line with its weight
		bestLine.Weight = currentWeight
		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSEV32(canvas, target, width, height)
			ssim := quickSSIMV32(canvas, target, width, height)
			meanBrightness := calculateMeanBrightnessV32(canvas, width, height)
			fmt.Printf("Progress: %d/%d lines (weight: %.1f, score: %.2f, MSE: %.1f, SSIM: %.4f, brightness: %.1f)\n",
				i+1, config.NumLines, currentWeight, bestLine.Score, mse, ssim, meanBrightness)
		}
	}

	fmt.Printf("\nPhase 1 complete: %d lines added\n", len(lines))
	initialSSIM := quickSSIMV32(canvas, target, width, height)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	// Phase 2: Simple heuristic removal (remove lines with negative contribution)
	fmt.Println("\n--- Phase 2: Heuristic Removal ---")
	lines = heuristicRemovalV32(lines, pins, canvas, target, width, height, config)

	return lines, canvas
}

// createImportanceMapV32 creates edge-weighted importance map
func createImportanceMapV32(edgeMap *image.Gray, width, height int, edgeWeight float64) [][]float64 {
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

// findBestLineV32 finds the best line from current pin using MSE-based scoring with importance weighting
func findBestLineV32(currentPin int, pins []Pin, canvas, target, importance [][]float64,
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

		// Calculate improvement score (MSE reduction with importance weighting)
		score := 0.0
		for _, p := range pixels {
			x, y := p[0], p[1]
			currentVal := canvas[y][x]
			targetVal := target[y][x]
			newVal := math.Max(0, currentVal-lineWeight)

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

// drawLineV32 draws a line on canvas with anti-aliasing
func drawLineV32(canvas [][]float64, from, to Pin, weight float64, width, height int) {
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

// heuristicRemovalV32 performs simple heuristic removal
func heuristicRemovalV32(lines []Line, pins []Pin, canvas, target [][]float64,
	width, height int, config *Config) []Line {

	// Sample 10% of lines for removal evaluation
	sampleSize := len(lines) / 10
	if sampleSize < 100 {
		sampleSize = min(100, len(lines))
	}

	// Evaluate each sampled line's contribution
	type LineContrib struct {
		index        int
		contribution float64
	}
	contribs := make([]LineContrib, 0)

	for i := 0; i < len(lines); i += len(lines) / sampleSize {
		// Remove line temporarily
		tempCanvas := copyCanvasV32(canvas, width, height)
		removeLineV32(tempCanvas, pins[lines[i].From], pins[lines[i].To], lines[i].Weight, width, height)

		// Calculate MSE without this line
		mseWithout := calculateMSEV32(tempCanvas, target, width, height)
		mseWith := calculateMSEV32(canvas, target, width, height)

		// Contribution is how much MSE increases when line is removed
		contribution := mseWithout - mseWith

		contribs = append(contribs, LineContrib{index: i, contribution: contribution})
	}

	// Sort by contribution (ascending)
	sort.Slice(contribs, func(i, j int) bool {
		return contribs[i].contribution < contribs[j].contribution
	})

	// Remove lines with negative contribution
	initialMSE := calculateMSEV32(canvas, target, width, height)
	removed := 0

	for _, lc := range contribs {
		if lc.contribution > 0 {
			break // Stop when we reach lines with positive contribution
		}

		// Remove line
		removeLineV32(canvas, pins[lines[lc.index].From], pins[lines[lc.index].To],
			lines[lc.index].Weight, width, height)

		newMSE := calculateMSEV32(canvas, target, width, height)
		if newMSE <= initialMSE {
			removed++
			lines[lc.index].Score = -1 // Mark for removal
		} else {
			// Restore line if MSE got worse
			drawLineV32(canvas, pins[lines[lc.index].From], pins[lines[lc.index].To],
				lines[lc.index].Weight, width, height)
		}
	}

	// Filter out removed lines
	filteredLines := make([]Line, 0)
	for _, line := range lines {
		if line.Score >= 0 {
			filteredLines = append(filteredLines, line)
		}
	}

	finalMSE := calculateMSEV32(canvas, target, width, height)
	finalSSIM := quickSSIMV32(canvas, target, width, height)
	fmt.Printf("Removed %d lines (MSE: %.1f -> %.1f, SSIM: %.4f)\n", removed, initialMSE, finalMSE, finalSSIM)

	return filteredLines
}

// Helper functions

func copyCanvasV32(canvas [][]float64, width, height int) [][]float64 {
	temp := make([][]float64, height)
	for y := 0; y < height; y++ {
		temp[y] = make([]float64, width)
		copy(temp[y], canvas[y])
	}
	return temp
}

func removeLineV32(canvas [][]float64, from, to Pin, weight float64, width, height int) {
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

func calculateMSEV32(canvas, target [][]float64, width, height int) float64 {
	sum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			sum += diff * diff
		}
	}
	return sum / float64(width*height)
}

func quickSSIMV32(canvas, target [][]float64, width, height int) float64 {
	// Simple SSIM approximation using local windows
	windowSize := 11
	c1 := 6.5025  // (0.01 * 255)^2
	c2 := 58.5225 // (0.03 * 255)^2

	ssimSum := 0.0
	count := 0

	for y := windowSize; y < height-windowSize; y += windowSize {
		for x := windowSize; x < width-windowSize; x += windowSize {
			// Calculate local statistics
			meanCanvas := 0.0
			meanTarget := 0.0
			varCanvas := 0.0
			varTarget := 0.0
			covar := 0.0
			n := 0

			for dy := -windowSize / 2; dy <= windowSize/2; dy++ {
				for dx := -windowSize / 2; dx <= windowSize/2; dx++ {
					px := x + dx
					py := y + dy
					if px >= 0 && px < width && py >= 0 && py < height {
						c := canvas[py][px]
						t := target[py][px]
						meanCanvas += c
						meanTarget += t
						n++
					}
				}
			}

			if n == 0 {
				continue
			}

			meanCanvas /= float64(n)
			meanTarget /= float64(n)

			for dy := -windowSize / 2; dy <= windowSize/2; dy++ {
				for dx := -windowSize / 2; dx <= windowSize/2; dx++ {
					px := x + dx
					py := y + dy
					if px >= 0 && px < width && py >= 0 && py < height {
						c := canvas[py][px] - meanCanvas
						t := target[py][px] - meanTarget
						varCanvas += c * c
						varTarget += t * t
						covar += c * t
					}
				}
			}

			varCanvas /= float64(n)
			varTarget /= float64(n)
			covar /= float64(n)

			// SSIM formula
			numerator := (2*meanCanvas*meanTarget + c1) * (2*covar + c2)
			denominator := (meanCanvas*meanCanvas + meanTarget*meanTarget + c1) *
				(varCanvas + varTarget + c2)

			if denominator > 0 {
				ssimSum += numerator / denominator
				count++
			}
		}
	}

	if count > 0 {
		return ssimSum / float64(count)
	}
	return 0.0
}

func calculateMeanBrightnessV32(canvas [][]float64, width, height int) float64 {
	sum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum += canvas[y][x]
		}
	}
	return sum / float64(width*height)
}
