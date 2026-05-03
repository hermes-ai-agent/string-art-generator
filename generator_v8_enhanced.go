package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV8Enhanced builds on the successful v5 approach with key improvements:
// 1. Better importance map with face-region detection
// 2. Add/Remove optimization - greedy add phase followed by removal pass
// 3. Enhanced perceptual scoring
// 4. Improved line removal strategy
func GenerateStringArtV8Enhanced(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v8.0 Enhanced (V5-based) ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	// Create enhanced importance map
	importance := createImportanceMapV8Enhanced(img, edgeMap, width, height)

	// Pre-compute line pixels with anti-aliasing (like v5)
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := centerX - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	// Initialize canvas
	canvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 255.0
		}
	}

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)

	// Phase 1: Greedy Add Phase with enhanced scoring
	fmt.Println("\n--- Phase 1: Enhanced Greedy Add Phase ---")
	recentScores := make([]float64, 0, 50)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV8Enhanced(currentPin, pins, canvas, img, edgeMap, importance,
			linePixels, config, usedLines, width, height)

		if bestLine.Score <= 0.001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Enhanced stagnation detection
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 50 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 50 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < 0.5 {
				stagnationCount++
				if stagnationCount > 20 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Apply line to canvas
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				canvas[pixel.Y][pixel.X] -= float64(config.LineWeight) * pixel.Weight
				if canvas[pixel.Y][pixel.X] < 0 {
					canvas[pixel.Y][pixel.X] = 0
				}
			}
		}

		usedLines[key]++
		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSEV5(canvas, img, width, height)
			ssim := quickSSIMV5(canvas, img, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM~%.3f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim)
		}
	}

	// Phase 2: Enhanced Add/Remove Optimization
	fmt.Println("\n--- Phase 2: Enhanced Add/Remove Optimization ---")
	lines = enhancedLineRemovalV8(lines, pins, canvas, img, importance, linePixels, config, width, height)

	// Final metrics
	finalMSE := calculateMSEV5(canvas, img, width, height)
	finalSSIM := quickSSIMV5(canvas, img, width, height)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	return lines, canvas
}

// createImportanceMapV8Enhanced creates enhanced importance map with face-region detection
func createImportanceMapV8Enhanced(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	centerX, centerY := float64(width)/2, float64(height)/2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	// Detect potential face region (center-upper area)
	faceRegionTop := int(float64(height) * 0.2)
	faceRegionBottom := int(float64(height) * 0.8)
	faceRegionLeft := int(float64(width) * 0.2)
	faceRegionRight := int(float64(width) * 0.8)

	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grayVal := float64(img.GrayAt(x, y).Y)
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)

			// 1. Darkness importance (darker = more important)
			darknessImportance := (255.0 - grayVal) / 255.0

			// 2. Edge importance
			edgeImportance := edgeVal / 255.0

			// 3. Face region detection - boost importance in potential face area (more conservative)
			faceBoost := 1.0
			if x >= faceRegionLeft && x <= faceRegionRight && y >= faceRegionTop && y <= faceRegionBottom {
				// Check if this looks like a face feature (dark areas in face region)
				if grayVal < 140 && edgeVal > 100 {
					faceBoost = 2.0 // Moderate boost for potential eyes, nose, mouth
				} else if grayVal < 160 && edgeVal > 60 {
					faceBoost = 1.6 // Light boost for face shadows
				} else if grayVal < 180 {
					faceBoost = 1.3 // Very light boost for general face area
				}
			}

			// 4. Center weighting - concentrate lines in center (like v5)
			ddx := float64(x) - centerX
			ddy := float64(y) - centerY
			dist := math.Sqrt(ddx*ddx+ddy*ddy) / maxDist
			// Moderate center weighting (like v5)
			centerWeight := math.Exp(-1.0 * dist * dist)

			// Combine: darkness primary, edges secondary (like v5)
			base := darknessImportance*0.6 + edgeImportance*0.4

			// Apply spatial and face weighting (more conservative)
			spatialWeight := 0.4 + 0.6*centerWeight
			importance[y][x] = base * spatialWeight * faceBoost

			// Minimum importance
			if importance[y][x] < 0.02 {
				importance[y][x] = 0.02
			}
		}
	}

	return importance
}

// findBestLineV8Enhanced finds the best next line using enhanced perceptual scoring
func findBestLineV8Enhanced(fromPin int, pins []Pin, canvas [][]float64, img *image.Gray, edgeMap *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config, usedLines map[[2]int]int,
	width, height int) Line {

	numPins := len(pins)

	type candidate struct {
		toPin int
		score float64
	}

	candidates := make([]candidate, 0, numPins)
	var mu sync.Mutex
	var wg sync.WaitGroup

	type evalTask struct {
		toPin int
		key   [2]int
	}

	tasks := make([]evalTask, 0, numPins)
	for toPin := 0; toPin < numPins; toPin++ {
		if toPin == fromPin {
			continue
		}
		key := [2]int{min(fromPin, toPin), max(fromPin, toPin)}

		distance := key[1] - key[0]
		if distance < config.MinDistance || numPins-distance < config.MinDistance {
			continue
		}

		if usedLines[key] >= 4 {
			continue
		}

		tasks = append(tasks, evalTask{toPin: toPin, key: key})
	}

	batchSize := (len(tasks) + config.Workers - 1) / config.Workers

	for w := 0; w < config.Workers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > len(tasks) {
			end = len(tasks)
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			localCandidates := make([]candidate, 0, end-start)

			for ti := start; ti < end; ti++ {
				task := tasks[ti]

				score := evaluateLineV8Enhanced(task.key, canvas, img, edgeMap, importance,
					linePixels, config, width, height)

				if usedLines[task.key] > 0 {
					score *= 1.0 / (1.0 + 0.4*float64(usedLines[task.key]))
				}

				if score > 0 {
					localCandidates = append(localCandidates, candidate{toPin: task.toPin, score: score})
				}
			}

			mu.Lock()
			candidates = append(candidates, localCandidates...)
			mu.Unlock()
		}(start, end)
	}

	wg.Wait()

	if len(candidates) == 0 {
		return Line{From: fromPin, To: -1, Score: 0}
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return Line{From: fromPin, To: best.toPin, Score: best.score}
}

// evaluateLineV8Enhanced evaluates a line using enhanced perceptual scoring
func evaluateLineV8Enhanced(key [2]int, canvas [][]float64, img *image.Gray, edgeMap *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config, width, height int) float64 {

	pixels := linePixels[key]
	if len(pixels) == 0 {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0

	for _, pixel := range pixels {
		if pixel.X < 0 || pixel.X >= width || pixel.Y < 0 || pixel.Y >= height {
			continue
		}

		currentVal := canvas[pixel.Y][pixel.X]
		targetVal := float64(img.GrayAt(pixel.X, pixel.Y).Y)
		edgeVal := float64(edgeMap.GrayAt(pixel.X, pixel.Y).Y)

		delta := float64(config.LineWeight) * pixel.Weight
		newVal := currentVal - delta
		if newVal < 0 {
			newVal = 0
		}

		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)
		improvement := oldError - newError

		imp := importance[pixel.Y][pixel.X]
		edgeBonus := 1.0 + edgeVal/255.0*config.EdgeWeight*0.2

		if improvement > 0 {
			totalScore += improvement * imp * edgeBonus * pixel.Weight
			improvingPixels++
		} else if improvement < 0 {
			// Mild over-darkening penalty (like v5)
			totalScore += improvement * imp * 1.1 * pixel.Weight
			worseningPixels++
		}
		totalWeight += imp * pixel.Weight
	}

	if totalWeight <= 0 {
		return 0
	}

	score := totalScore / totalWeight

	// Worsen ratio penalty (like v5)
	totalPixels := improvingPixels + worseningPixels
	if totalPixels > 0 {
		worsenRatio := float64(worseningPixels) / float64(totalPixels)
		if worsenRatio > 0.45 {
			score *= (1.0 - worsenRatio*0.5)
		}
	}

	return score
}

// enhancedLineRemovalV8 implements enhanced line removal optimization
func enhancedLineRemovalV8(lines []Line, pins []Pin, canvas [][]float64, img *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config, width, height int) []Line {

	if len(lines) == 0 {
		return lines
	}

	maxRemovals := len(lines) / 5 // Allow removing up to 1/5 of lines

	currentMSE := calculateMSEV5(canvas, img, width, height)
	currentSSIM := quickSSIMV5(canvas, img, width, height)
	fmt.Printf("Before enhanced removal: MSE = %.2f, SSIM~%.3f\n", currentMSE, currentSSIM)

	type removalCandidate struct {
		index       int
		improvement float64
	}

	candidates := make([]removalCandidate, 0)

	// Evaluate each line for potential removal
	for idx, line := range lines {
		if line.From < 0 || line.To < 0 {
			continue
		}

		key := [2]int{min(line.From, line.To), max(line.From, line.To)}
		pixels := linePixels[key]

		// Calculate error improvement from removal (lightening)
		errorChange := 0.0
		for _, pixel := range pixels {
			if pixel.X < 0 || pixel.X >= width || pixel.Y < 0 || pixel.Y >= height {
				continue
			}

			currentVal := canvas[pixel.Y][pixel.X]
			targetVal := float64(img.GrayAt(pixel.X, pixel.Y).Y)

			delta := float64(config.LineWeight) * pixel.Weight
			newVal := currentVal + delta
			if newVal > 255 {
				newVal = 255
			}

			oldErr := (currentVal - targetVal) * (currentVal - targetVal)
			newErr := (newVal - targetVal) * (newVal - targetVal)
			imp := importance[pixel.Y][pixel.X]
			errorChange += (oldErr - newErr) * imp * pixel.Weight
		}

		if errorChange > 0 {
			candidates = append(candidates, removalCandidate{index: idx, improvement: errorChange})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].improvement > candidates[j].improvement
	})

	removedIndices := make(map[int]bool)
	removedCount := 0

	// Remove lines that improve quality
	for _, c := range candidates {
		if removedCount >= maxRemovals {
			break
		}

		line := lines[c.index]
		key := [2]int{min(line.From, line.To), max(line.From, line.To)}
		pixels := linePixels[key]

		// Remove line from canvas
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				delta := float64(config.LineWeight) * pixel.Weight
				canvas[pixel.Y][pixel.X] += delta
				if canvas[pixel.Y][pixel.X] > 255 {
					canvas[pixel.Y][pixel.X] = 255
				}
			}
		}

		removedIndices[c.index] = true
		removedCount++
	}

	newLines := make([]Line, 0, len(lines)-removedCount)
	for idx, line := range lines {
		if !removedIndices[idx] {
			newLines = append(newLines, line)
		}
	}

	newMSE := calculateMSEV5(canvas, img, width, height)
	newSSIM := quickSSIMV5(canvas, img, width, height)
	fmt.Printf("After enhanced removal: MSE = %.2f, SSIM~%.3f (removed %d lines)\n", newMSE, newSSIM, removedCount)

	return newLines
}

// Helper functions for v5-style calculations
func calculateMSEV5(canvas [][]float64, img *image.Gray, width, height int) float64 {
	mse := 0.0
	count := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - float64(img.GrayAt(x, y).Y)
			mse += diff * diff
			count++
		}
	}
	return mse / float64(count)
}

func quickSSIMV5(canvas [][]float64, img *image.Gray, width, height int) float64 {
	// Simple SSIM approximation
	meanCanvas := 0.0
	meanImg := 0.0
	count := 0
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			meanCanvas += canvas[y][x]
			meanImg += float64(img.GrayAt(x, y).Y)
			count++
		}
	}
	meanCanvas /= float64(count)
	meanImg /= float64(count)
	
	varCanvas := 0.0
	varImg := 0.0
	covar := 0.0
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diffCanvas := canvas[y][x] - meanCanvas
			diffImg := float64(img.GrayAt(x, y).Y) - meanImg
			varCanvas += diffCanvas * diffCanvas
			varImg += diffImg * diffImg
			covar += diffCanvas * diffImg
		}
	}
	
	varCanvas /= float64(count)
	varImg /= float64(count)
	covar /= float64(count)
	
	c1 := 6.5025  // (0.01*255)^2
	c2 := 58.5225 // (0.03*255)^2
	
	numerator := (2*meanCanvas*meanImg + c1) * (2*covar + c2)
	denominator := (meanCanvas*meanCanvas + meanImg*meanImg + c1) * (varCanvas + varImg + c2)
	
	if denominator == 0 {
		return 1.0
	}
	
	return numerator / denominator
}