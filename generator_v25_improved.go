package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV25Improved - Focused improvements over v5:
// 1. Enhanced face detection with better center weighting
// 2. Perceptual line scoring (local contrast + error reduction)
// 3. More aggressive line removal with lower threshold
// 4. Better adaptive weight curve
func GenerateStringArtV25Improved(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

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

	// Create enhanced importance map
	importance := createEnhancedImportanceMapV25(img, edgeMap, width, height)

	// Pre-compute line pixels
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)

	fmt.Printf("=== String Art Generator v25 (Improved) ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Adaptive parameters
	baseWeight := float64(config.LineWeight)
	recentScores := make([]float64, 0, 20)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		// Better adaptive weight curve: slower decay for better detail
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.4*progress) // Start at 100%, end at 60%
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV25(currentPin, pins, canvas, target, edgeMap, importance,
			linePixels, config, adaptiveWeight, usedLines)

		if bestLine.Score <= 0.1 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Adaptive stopping
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 20 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 20 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < config.StopThreshold*0.3 {
				stagnationCount++
				if stagnationCount > 3 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Draw line on canvas
		drawLineAAOnCanvas(canvas, pins[bestLine.From], pins[bestLine.To], adaptiveWeight, width, height)

		// Track usage
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSE(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, adaptiveWeight)
		}
	}

	// More aggressive line removal pass
	fmt.Println("\n--- Enhanced Line Removal Pass ---")
	lines, canvas = enhancedLineRemovalPassV25(lines, pins, target, canvas, importance, config, width, height)

	fmt.Printf("\nFinal: %d lines, MSE: %.1f\n", len(lines), calculateMSE(canvas, target, width, height))
	return lines, canvas
}

// createEnhancedImportanceMapV25 creates a better importance map with face detection
func createEnhancedImportanceMapV25(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	centerX, centerY := float64(width)/2, float64(height)/2

	// Detect face region (darker areas in upper-center)
	faceRegion := detectFaceRegionV25(img, width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Base importance from edges
			edgeVal := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			baseImportance := 1.0 + edgeVal*2.0

			// Distance from center (favor center)
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			distFromCenter := math.Sqrt(dx*dx + dy*dy)
			maxDist := math.Sqrt(centerX*centerX + centerY*centerY)
			centerWeight := 1.0 + (1.0-distFromCenter/maxDist)*0.5

			// Face region boost
			faceWeight := 1.0
			if faceRegion[y][x] {
				faceWeight = 2.5 // Strong boost for face features
			}

			importance[y][x] = baseImportance * centerWeight * faceWeight
		}
	}

	return importance
}

// detectFaceRegionV25 detects likely face region (upper-center dark areas)
func detectFaceRegionV25(img *image.Gray, width, height int) [][]bool {
	faceRegion := make([][]bool, height)
	for y := 0; y < height; y++ {
		faceRegion[y] = make([]bool, width)
	}

	// Calculate average brightness
	totalBrightness := 0.0
	count := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			totalBrightness += float64(img.GrayAt(x, y).Y)
			count++
		}
	}
	avgBrightness := totalBrightness / float64(count)

	// Face is typically in upper 60% and center 70%
	centerX := width / 2
	faceTop := height / 5
	faceBottom := height * 3 / 5
	faceLeft := width * 15 / 100
	faceRight := width * 85 / 100

	for y := faceTop; y < faceBottom; y++ {
		for x := faceLeft; x < faceRight; x++ {
			brightness := float64(img.GrayAt(x, y).Y)
			
			// Face features are darker than average
			if brightness < avgBrightness*0.85 {
				// Distance from center
				dx := float64(x - centerX)
				dy := float64(y - (faceTop+faceBottom)/2)
				dist := math.Sqrt(dx*dx + dy*dy)
				
				// Elliptical face region
				maxDist := float64(width) * 0.25
				if dist < maxDist {
					faceRegion[y][x] = true
				}
			}
		}
	}

	return faceRegion
}

// findBestLineV25 finds the best line with perceptual scoring
func findBestLineV25(currentPin int, pins []Pin, canvas, target [][]float64, edgeMap *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel,
	config *Config, weight float64, usedLines map[[2]int]int) Line {

	type candidate struct {
		to    int
		score float64
	}

	numPins := len(pins)
	candidates := make([]candidate, 0, numPins)

	// Parallel evaluation
	var mu sync.Mutex
	var wg sync.WaitGroup
	chunkSize := numPins / config.Workers
	if chunkSize < 1 {
		chunkSize = 1
	}

	for w := 0; w < config.Workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := workerID * chunkSize
			end := start + chunkSize
			if workerID == config.Workers-1 {
				end = numPins
			}

			localCandidates := make([]candidate, 0)

			for to := start; to < end; to++ {
				if to == currentPin {
					continue
				}

				// Check distance constraint
				distance := abs(to - currentPin)
				if distance < config.MinDistance || numPins-distance < config.MinDistance {
					continue
				}

				// Get line pixels
				key := [2]int{min(currentPin, to), max(currentPin, to)}
				pixels, exists := linePixels[key]
				if !exists || len(pixels) == 0 {
					continue
				}

				// Calculate perceptual score
				score := calculatePerceptualScoreV25(pixels, canvas, target, importance, weight)

				// Penalize overused lines
				usageCount := usedLines[key]
				if usageCount > 0 {
					score *= math.Pow(0.85, float64(usageCount))
				}

				if score > 0 {
					localCandidates = append(localCandidates, candidate{to: to, score: score})
				}
			}

			mu.Lock()
			candidates = append(candidates, localCandidates...)
			mu.Unlock()
		}(w)
	}

	wg.Wait()

	if len(candidates) == 0 {
		return Line{From: currentPin, To: currentPin, Score: 0}
	}

	// Find best candidate
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return Line{From: currentPin, To: candidates[0].to, Score: candidates[0].score}
}

// calculatePerceptualScoreV25 calculates perceptual quality improvement
func calculatePerceptualScoreV25(pixels []AntiAliasedPixel, canvas, target, importance [][]float64, weight float64) float64 {
	totalScore := 0.0

	for _, p := range pixels {
		currentVal := canvas[p.Y][p.X]
		targetVal := target[p.Y][p.X]

		// Simulate adding the line
		newVal := currentVal - weight*p.Weight
		if newVal < 0 {
			newVal = 0
		}

		// Error reduction
		oldError := math.Abs(currentVal - targetVal)
		newError := math.Abs(newVal - targetVal)
		errorReduction := oldError - newError

		// Only count if we're improving (moving closer to target)
		if errorReduction > 0 {
			imp := importance[p.Y][p.X]
			totalScore += errorReduction * imp * p.Weight
		}
	}

	return totalScore
}

// enhancedLineRemovalPassV25 removes lines that hurt quality (more aggressive)
func enhancedLineRemovalPassV25(lines []Line, pins []Pin, target, canvas [][]float64,
	importance [][]float64, config *Config, width, height int) ([]Line, [][]float64) {

	if len(lines) == 0 {
		return lines, canvas
	}

	removedCount := 0
	maxRemovals := len(lines) / 8 // Remove up to 12.5% of lines (more aggressive)

	currentError := calculateWeightedMSE(canvas, target, importance, width, height)
	fmt.Printf("Before removal: weighted MSE = %.2f\n", currentError)

	type removalCandidate struct {
		index       int
		improvement float64
	}

	candidates := make([]removalCandidate, 0)

	for idx, line := range lines {
		if line.From < 0 || line.To < 0 {
			continue
		}

		pixels := getAntiAliasedLinePixels(pins[line.From], pins[line.To], width, height)

		// Calculate error change if we remove this line
		errorChange := 0.0
		for _, p := range pixels {
			currentVal := canvas[p.Y][p.X]
			targetVal := target[p.Y][p.X]

			// Simulate removal (lighten)
			restoredVal := currentVal + float64(config.LineWeight)*p.Weight*0.7
			if restoredVal > 255 {
				restoredVal = 255
			}

			oldErr := (currentVal - targetVal) * (currentVal - targetVal)
			newErr := (restoredVal - targetVal) * (restoredVal - targetVal)

			imp := importance[p.Y][p.X]
			errorChange += (oldErr - newErr) * imp
		}

		// Lower threshold for removal (more aggressive)
		if errorChange > -50 { // Accept even small improvements or neutral changes
			candidates = append(candidates, removalCandidate{index: idx, improvement: errorChange})
		}
	}

	// Sort by improvement
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].improvement > candidates[j].improvement
	})

	// Remove the worst lines
	removedIndices := make(map[int]bool)
	for _, c := range candidates {
		if removedCount >= maxRemovals {
			break
		}

		line := lines[c.index]
		pixels := getAntiAliasedLinePixels(pins[line.From], pins[line.To], width, height)

		// Actually remove
		for _, p := range pixels {
			canvas[p.Y][p.X] += float64(config.LineWeight) * p.Weight * 0.7
			if canvas[p.Y][p.X] > 255 {
				canvas[p.Y][p.X] = 255
			}
		}

		removedIndices[c.index] = true
		removedCount++
	}

	// Build new line list
	newLines := make([]Line, 0, len(lines)-removedCount)
	for idx, line := range lines {
		if !removedIndices[idx] {
			newLines = append(newLines, line)
		}
	}

	newError := calculateWeightedMSE(canvas, target, importance, width, height)
	fmt.Printf("After removal: weighted MSE = %.2f (removed %d lines)\n", newError, removedCount)

	return newLines, canvas
}
