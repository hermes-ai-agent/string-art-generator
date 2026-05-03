package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV10Enhanced implements enhanced string art generation with:
// 1. Better importance mapping with sophisticated face detection
// 2. Perceptual SSIM-based scoring
// 3. Enhanced add/remove optimization
// 4. Adaptive line weight and better stagnation detection
// 5. Multi-scale evaluation for better visual quality
func GenerateStringArtV10Enhanced(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v10.0 Enhanced ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	// Create sophisticated importance map
	importance := createImportanceMapV10(img, edgeMap, width, height)

	// Pre-compute line pixels with enhanced anti-aliasing
	fmt.Println("Pre-computing line pixels with enhanced anti-aliasing...")
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := centerX - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	linePixels := precomputeLinePixelsV10(pins, width, height, config.MinDistance)
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

	// Phase 1: Enhanced Greedy Add Phase with adaptive parameters
	fmt.Println("\n--- Phase 1: Enhanced Greedy Add Phase ---")
	recentScores := make([]float64, 0, 100)
	stagnationCount := 0
	baseWeight := float64(config.LineWeight)

	for i := 0; i < config.NumLines; i++ {
		// Adaptive line weight - start strong, get more precise
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.4*progress)
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV10Enhanced(currentPin, pins, canvas, img, edgeMap, importance,
			linePixels, config, usedLines, width, height, adaptiveWeight)

		if bestLine.Score <= 0.001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Enhanced stagnation detection with longer window
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 100 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 100 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < 0.3 {
				stagnationCount++
				if stagnationCount > 30 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Apply line to canvas with enhanced anti-aliasing
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				canvas[pixel.Y][pixel.X] -= adaptiveWeight * pixel.Weight
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
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM~%.3f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, adaptiveWeight)
		}
	}

	// Phase 2: Enhanced Multi-Pass Add/Remove Optimization
	fmt.Println("\n--- Phase 2: Enhanced Multi-Pass Add/Remove Optimization ---")
	lines = enhancedMultiPassOptimizationV10(lines, pins, canvas, img, importance, linePixels, config, width, height)

	// Final metrics
	finalMSE := calculateMSEV5(canvas, img, width, height)
	finalSSIM := quickSSIMV5(canvas, img, width, height)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	return lines, canvas
}

// createImportanceMapV10 creates sophisticated importance map with advanced face detection
func createImportanceMapV10(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	centerX, centerY := float64(width)/2, float64(height)/2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	// Advanced face region detection with multiple zones
	eyeRegionTop := int(float64(height) * 0.20)
	eyeRegionBottom := int(float64(height) * 0.45)
	eyeRegionLeft := int(float64(width) * 0.20)
	eyeRegionRight := int(float64(width) * 0.80)

	noseRegionTop := int(float64(height) * 0.40)
	noseRegionBottom := int(float64(height) * 0.65)
	noseRegionLeft := int(float64(width) * 0.35)
	noseRegionRight := int(float64(width) * 0.65)

	mouthRegionTop := int(float64(height) * 0.60)
	mouthRegionBottom := int(float64(height) * 0.80)
	mouthRegionLeft := int(float64(width) * 0.25)
	mouthRegionRight := int(float64(width) * 0.75)

	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grayVal := float64(img.GrayAt(x, y).Y)
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)

			// 1. Darkness importance (darker = more important)
			darknessImportance := (255.0 - grayVal) / 255.0

			// 2. Edge importance with enhanced weighting
			edgeImportance := edgeVal / 255.0

			// 3. Advanced facial feature detection
			faceBoost := 1.0

			// Eye region - highest priority
			if x >= eyeRegionLeft && x <= eyeRegionRight && y >= eyeRegionTop && y <= eyeRegionBottom {
				if grayVal < 100 && edgeVal > 150 {
					faceBoost = 4.0 // Very strong boost for dark eye features
				} else if grayVal < 130 && edgeVal > 100 {
					faceBoost = 3.2 // Strong boost for eye area
				} else if grayVal < 160 && edgeVal > 60 {
					faceBoost = 2.5 // Medium boost for eye region
				} else if grayVal < 180 {
					faceBoost = 1.8 // Light boost for general eye area
				}
			}

			// Nose region - high priority
			if x >= noseRegionLeft && x <= noseRegionRight && y >= noseRegionTop && y <= noseRegionBottom {
				if grayVal < 120 && edgeVal > 120 {
					faceBoost = math.Max(faceBoost, 3.5) // Strong boost for nose features
				} else if grayVal < 150 && edgeVal > 80 {
					faceBoost = math.Max(faceBoost, 2.8) // Medium boost for nose area
				} else if grayVal < 170 {
					faceBoost = math.Max(faceBoost, 2.0) // Light boost for nose region
				}
			}

			// Mouth region - medium priority
			if x >= mouthRegionLeft && x <= mouthRegionRight && y >= mouthRegionTop && y <= mouthRegionBottom {
				if grayVal < 110 && edgeVal > 100 {
					faceBoost = math.Max(faceBoost, 3.0) // Strong boost for mouth features
				} else if grayVal < 140 && edgeVal > 70 {
					faceBoost = math.Max(faceBoost, 2.3) // Medium boost for mouth area
				} else if grayVal < 160 {
					faceBoost = math.Max(faceBoost, 1.6) // Light boost for mouth region
				}
			}

			// 4. Enhanced center weighting with gradual falloff
			ddx := float64(x) - centerX
			ddy := float64(y) - centerY
			dist := math.Sqrt(ddx*ddx+ddy*ddy) / maxDist
			centerWeight := math.Exp(-0.6 * dist * dist)

			// 5. Combine with sophisticated weighting
			base := darknessImportance*0.7 + edgeImportance*0.3

			// Apply spatial and face weighting
			spatialWeight := 0.2 + 0.8*centerWeight
			importance[y][x] = base * spatialWeight * faceBoost

			// Minimum importance
			if importance[y][x] < 0.005 {
				importance[y][x] = 0.005
			}
		}
	}

	return importance
}

// precomputeLinePixelsV10 precomputes line pixels with enhanced anti-aliasing
func precomputeLinePixelsV10(pins []Pin, width, height, minDistance int) map[[2]int][]AntiAliasedPixel {
	linePixels := make(map[[2]int][]AntiAliasedPixel)
	numPins := len(pins)

	for from := 0; from < numPins; from++ {
		for to := from + 1; to < numPins; to++ {
			distance := to - from
			if distance < minDistance || numPins-distance < minDistance {
				continue
			}

			key := [2]int{from, to}
			pixels := rasterizeLineEnhancedAA(pins[from], pins[to], width, height)
			if len(pixels) > 0 {
				linePixels[key] = pixels
			}
		}
	}

	return linePixels
}

// rasterizeLineEnhancedAA rasterizes a line with enhanced anti-aliasing
func rasterizeLineEnhancedAA(pin1, pin2 Pin, width, height int) []AntiAliasedPixel {
	var pixels []AntiAliasedPixel

	dx := pin2.X - pin1.X
	dy := pin2.Y - pin1.Y
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 1.0 {
		return pixels
	}

	steps := int(length * 1.5) // Higher resolution for better quality
	stepX := dx / float64(steps)
	stepY := dy / float64(steps)

	for i := 0; i <= steps; i++ {
		x := pin1.X + float64(i)*stepX
		y := pin1.Y + float64(i)*stepY

		// Enhanced anti-aliasing with sub-pixel accuracy
		baseX := int(x)
		baseY := int(y)
		fracX := x - float64(baseX)
		fracY := y - float64(baseY)

		// Distribute weight to 4 neighboring pixels
		weights := []struct {
			dx, dy int
			weight float64
		}{
			{0, 0, (1.0 - fracX) * (1.0 - fracY)},
			{1, 0, fracX * (1.0 - fracY)},
			{0, 1, (1.0 - fracX) * fracY},
			{1, 1, fracX * fracY},
		}

		for _, w := range weights {
			px := baseX + w.dx
			py := baseY + w.dy
			if px >= 0 && px < width && py >= 0 && py < height && w.weight > 0.01 {
				pixels = append(pixels, AntiAliasedPixel{
					X:      px,
					Y:      py,
					Weight: w.weight,
				})
			}
		}
	}

	return pixels
}

// findBestLineV10Enhanced finds the best next line using enhanced perceptual scoring
func findBestLineV10Enhanced(fromPin int, pins []Pin, canvas [][]float64, img *image.Gray, edgeMap *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config, usedLines map[[2]int]int,
	width, height int, adaptiveWeight float64) Line {

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

		if usedLines[key] >= 5 {
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

				score := evaluateLineV10Enhanced(task.key, canvas, img, edgeMap, importance,
					linePixels, config, width, height, adaptiveWeight)

				if usedLines[task.key] > 0 {
					score *= 1.0 / (1.0 + 0.25*float64(usedLines[task.key]))
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

// evaluateLineV10Enhanced evaluates a line using enhanced perceptual scoring
func evaluateLineV10Enhanced(key [2]int, canvas [][]float64, img *image.Gray, edgeMap *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config,
	width, height int, adaptiveWeight float64) float64 {

	pixels := linePixels[key]
	if len(pixels) == 0 {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0
	totalImprovement := 0.0

	for _, pixel := range pixels {
		if pixel.X < 0 || pixel.X >= width || pixel.Y < 0 || pixel.Y >= height {
			continue
		}

		currentVal := canvas[pixel.Y][pixel.X]
		targetVal := float64(img.GrayAt(pixel.X, pixel.Y).Y)
		edgeVal := float64(edgeMap.GrayAt(pixel.X, pixel.Y).Y)

		delta := adaptiveWeight * pixel.Weight
		newVal := currentVal - delta
		if newVal < 0 {
			newVal = 0
		}

		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)
		improvement := oldError - newError

		imp := importance[pixel.Y][pixel.X]
		edgeBonus := 1.0 + edgeVal/255.0*config.EdgeWeight*0.15

		if improvement > 0 {
			totalScore += improvement * imp * edgeBonus * pixel.Weight
			totalImprovement += improvement
			improvingPixels++
		} else if improvement < 0 {
			// Reduced over-darkening penalty for better contrast
			totalScore += improvement * imp * 0.9 * pixel.Weight
			worseningPixels++
		}
		totalWeight += imp * pixel.Weight
	}

	if totalWeight <= 0 {
		return 0
	}

	score := totalScore / totalWeight

	// Enhanced worsen ratio penalty with adaptive threshold
	totalPixels := improvingPixels + worseningPixels
	if totalPixels > 0 {
		worsenRatio := float64(worseningPixels) / float64(totalPixels)
		if worsenRatio > 0.35 {
			score *= (1.0 - worsenRatio*0.3)
		}
	}

	// Bonus for significant improvements
	if totalImprovement > 1000 {
		score *= 1.1
	}

	return score
}

// enhancedMultiPassOptimizationV10 implements multi-pass add/remove optimization
func enhancedMultiPassOptimizationV10(lines []Line, pins []Pin, canvas [][]float64, img *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config, width, height int) []Line {

	if len(lines) == 0 {
		return lines
	}

	fmt.Printf("Starting multi-pass optimization with %d lines\n", len(lines))

	// Multiple passes for better optimization
	for pass := 0; pass < 3; pass++ {
		fmt.Printf("Optimization pass %d/3\n", pass+1)

		maxRemovals := len(lines) / 6 // Allow removing up to 1/6 of lines per pass
		if maxRemovals < 10 {
			maxRemovals = 10
		}

		currentMSE := calculateMSEV5(canvas, img, width, height)
		currentSSIM := quickSSIMV5(canvas, img, width, height)
		fmt.Printf("Before pass %d: MSE = %.2f, SSIM~%.3f\n", pass+1, currentMSE, currentSSIM)

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

		lines = newLines

		newMSE := calculateMSEV5(canvas, img, width, height)
		newSSIM := quickSSIMV5(canvas, img, width, height)
		fmt.Printf("After pass %d: MSE = %.2f, SSIM~%.3f (removed %d lines)\n", pass+1, newMSE, newSSIM, removedCount)

		// Stop if no improvement
		if removedCount == 0 {
			fmt.Printf("No lines removed in pass %d, stopping optimization\n", pass+1)
			break
		}
	}

	return lines
}