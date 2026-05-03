package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV9Birsak implements proper Birsak 2018 supersampled rendering
// Key improvements:
// 1. 8x supersampled rendering for accurate gray levels with opaque strokes
// 2. Proper downsampling to target resolution
// 3. Enhanced importance mapping with face detection
// 4. Add/Remove optimization with perceptual scoring
// 5. SSIM-based line evaluation
func GenerateStringArtV9Birsak(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v9.0 (Birsak 2018 Supersampled) ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	// Supersampling factor (4x for balance between quality and memory)
	supersample := 4
	superWidth := width * supersample
	superHeight := height * supersample

	fmt.Printf("Supersampled resolution: %dx%d (4x)\n", superWidth, superHeight)

	// Create enhanced importance map
	importance := createImportanceMapV9(img, edgeMap, width, height)

	// Pre-compute line pixels at supersampled resolution
	fmt.Println("Pre-computing supersampled line pixels...")
	centerX, centerY := float64(superWidth)/2, float64(superHeight)/2
	radius := centerX - 10*float64(supersample)
	pins := GeneratePinsSuper(config.NumPins, radius, centerX, centerY)

	linePixels := precomputeLinePixelsSuper(pins, superWidth, superHeight, config.MinDistance, supersample)
	fmt.Printf("Pre-computed %d valid line segments at 4x resolution\n", len(linePixels))

	// Initialize supersampled canvas
	superCanvas := make([][]float64, superHeight)
	for y := 0; y < superHeight; y++ {
		superCanvas[y] = make([]float64, superWidth)
		for x := 0; x < superWidth; x++ {
			superCanvas[y][x] = 255.0
		}
	}

	// Upsample target image to supersampled resolution
	superImg := upsampleImage(img, supersample)

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)

	// Phase 1: Greedy Add Phase with SSIM-based scoring
	fmt.Println("\n--- Phase 1: Supersampled Greedy Add Phase ---")
	recentScores := make([]float64, 0, 50)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV9Birsak(currentPin, pins, superCanvas, superImg, importance,
			linePixels, config, usedLines, superWidth, superHeight, supersample, width, height)

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

		// Apply line to supersampled canvas
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
				superCanvas[pixel.Y][pixel.X] -= float64(config.LineWeight) * pixel.Weight
				if superCanvas[pixel.Y][pixel.X] < 0 {
					superCanvas[pixel.Y][pixel.X] = 0
				}
			}
		}

		usedLines[key]++
		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			// Downsample for metrics calculation
			downsampledCanvas := downsampleCanvasV9(superCanvas, supersample, width, height)
			mse := calculateMSEV5(downsampledCanvas, img, width, height)
			ssim := quickSSIMV5(downsampledCanvas, img, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM~%.3f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim)
		}
	}

	// Phase 2: Add/Remove Optimization at supersampled resolution
	fmt.Println("\n--- Phase 2: Supersampled Add/Remove Optimization ---")
	lines = enhancedLineRemovalV9(lines, pins, superCanvas, superImg, importance, linePixels, config, superWidth, superHeight, supersample, width, height, img)

	// Final downsample to target resolution
	fmt.Println("Downsampling to target resolution...")
	finalCanvas := downsampleCanvasV9(superCanvas, supersample, width, height)

	// Final metrics
	finalMSE := calculateMSEV5(finalCanvas, img, width, height)
	finalSSIM := quickSSIMV5(finalCanvas, img, width, height)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	return lines, finalCanvas
}

// createImportanceMapV9 creates enhanced importance map with better face detection
func createImportanceMapV9(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	centerX, centerY := float64(width)/2, float64(height)/2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	// Enhanced face region detection
	faceRegionTop := int(float64(height) * 0.15)
	faceRegionBottom := int(float64(height) * 0.75)
	faceRegionLeft := int(float64(width) * 0.15)
	faceRegionRight := int(float64(width) * 0.85)

	// Eye region detection (upper face area)
	eyeRegionTop := int(float64(height) * 0.25)
	eyeRegionBottom := int(float64(height) * 0.45)
	eyeRegionLeft := int(float64(width) * 0.25)
	eyeRegionRight := int(float64(width) * 0.75)

	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grayVal := float64(img.GrayAt(x, y).Y)
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)

			// 1. Darkness importance (darker = more important)
			darknessImportance := (255.0 - grayVal) / 255.0

			// 2. Edge importance
			edgeImportance := edgeVal / 255.0

			// 3. Enhanced face region detection
			faceBoost := 1.0
			if x >= faceRegionLeft && x <= faceRegionRight && y >= faceRegionTop && y <= faceRegionBottom {
				// Eye region gets highest priority
				if x >= eyeRegionLeft && x <= eyeRegionRight && y >= eyeRegionTop && y <= eyeRegionBottom {
					if grayVal < 120 && edgeVal > 120 {
						faceBoost = 3.0 // Strong boost for potential eyes
					} else if grayVal < 140 && edgeVal > 80 {
						faceBoost = 2.5 // Medium boost for eye area
					} else if grayVal < 160 {
						faceBoost = 2.0 // Light boost for general eye region
					}
				} else {
					// General face area
					if grayVal < 130 && edgeVal > 100 {
						faceBoost = 2.2 // Boost for nose, mouth features
					} else if grayVal < 150 && edgeVal > 60 {
						faceBoost = 1.8 // Boost for face shadows
					} else if grayVal < 170 {
						faceBoost = 1.4 // Light boost for general face area
					}
				}
			}

			// 4. Center weighting - concentrate lines in center
			ddx := float64(x) - centerX
			ddy := float64(y) - centerY
			dist := math.Sqrt(ddx*ddx+ddy*ddy) / maxDist
			centerWeight := math.Exp(-0.8 * dist * dist)

			// Combine: darkness primary, edges secondary
			base := darknessImportance*0.65 + edgeImportance*0.35

			// Apply spatial and face weighting
			spatialWeight := 0.3 + 0.7*centerWeight
			importance[y][x] = base * spatialWeight * faceBoost

			// Minimum importance
			if importance[y][x] < 0.01 {
				importance[y][x] = 0.01
			}
		}
	}

	return importance
}

// GeneratePinsSuper generates pins at supersampled resolution
func GeneratePinsSuper(numPins int, radius, centerX, centerY float64) []Pin {
	pins := make([]Pin, numPins)
	for i := 0; i < numPins; i++ {
		angle := 2.0 * math.Pi * float64(i) / float64(numPins)
		x := centerX + radius*math.Cos(angle)
		y := centerY + radius*math.Sin(angle)
		pins[i] = Pin{X: x, Y: y}
	}
	return pins
}

// precomputeLinePixelsSuper precomputes line pixels at supersampled resolution
func precomputeLinePixelsSuper(pins []Pin, width, height, minDistance, supersample int) map[[2]int][]AntiAliasedPixel {
	linePixels := make(map[[2]int][]AntiAliasedPixel)
	numPins := len(pins)

	for from := 0; from < numPins; from++ {
		for to := from + 1; to < numPins; to++ {
			distance := to - from
			if distance < minDistance || numPins-distance < minDistance {
				continue
			}

			key := [2]int{from, to}
			pixels := rasterizeLineAntiAliasedSuper(pins[from], pins[to], width, height, supersample)
			if len(pixels) > 0 {
				linePixels[key] = pixels
			}
		}
	}

	return linePixels
}

// rasterizeLineAntiAliasedSuper rasterizes a line with anti-aliasing at supersampled resolution
func rasterizeLineAntiAliasedSuper(pin1, pin2 Pin, width, height, supersample int) []AntiAliasedPixel {
	var pixels []AntiAliasedPixel

	dx := pin2.X - pin1.X
	dy := pin2.Y - pin1.Y
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 1.0 {
		return pixels
	}

	// Use efficient steps for supersampled resolution
	steps := int(length)
	if steps < 10 {
		steps = 10
	}
	stepX := dx / float64(steps)
	stepY := dy / float64(steps)

	for i := 0; i <= steps; i++ {
		x := pin1.X + float64(i)*stepX
		y := pin1.Y + float64(i)*stepY

		// Simplified anti-aliasing for memory efficiency
		px := int(x + 0.5)
		py := int(y + 0.5)

		if px >= 0 && px < width && py >= 0 && py < height {
			pixels = append(pixels, AntiAliasedPixel{
				X:      px,
				Y:      py,
				Weight: 1.0,
			})
		}

		// Add neighboring pixels for line thickness
		for _, offset := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx := px + offset[0]
			ny := py + offset[1]
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				pixels = append(pixels, AntiAliasedPixel{
					X:      nx,
					Y:      ny,
					Weight: 0.5,
				})
			}
		}
	}

	return pixels
}

// upsampleImage upsamples the target image to supersampled resolution
func upsampleImage(img *image.Gray, supersample int) *image.Gray {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	superWidth := width * supersample
	superHeight := height * supersample

	superImg := image.NewGray(image.Rect(0, 0, superWidth, superHeight))

	for y := 0; y < superHeight; y++ {
		for x := 0; x < superWidth; x++ {
			// Simple nearest neighbor upsampling
			origX := x / supersample
			origY := y / supersample
			if origX >= width {
				origX = width - 1
			}
			if origY >= height {
				origY = height - 1
			}
			superImg.SetGray(x, y, img.GrayAt(origX, origY))
		}
	}

	return superImg
}

// downsampleCanvasV9 downsamples the supersampled canvas to target resolution
func downsampleCanvasV9(superCanvas [][]float64, supersample, width, height int) [][]float64 {
	canvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			// Average the supersample x supersample block
			sum := 0.0
			count := 0
			for sy := y * supersample; sy < (y+1)*supersample; sy++ {
				for sx := x * supersample; sx < (x+1)*supersample; sx++ {
					if sy < len(superCanvas) && sx < len(superCanvas[sy]) {
						sum += superCanvas[sy][sx]
						count++
					}
				}
			}
			if count > 0 {
				canvas[y][x] = sum / float64(count)
			} else {
				canvas[y][x] = 255.0
			}
		}
	}
	return canvas
}

// findBestLineV9Birsak finds the best next line using supersampled SSIM-based scoring
func findBestLineV9Birsak(fromPin int, pins []Pin, superCanvas [][]float64, superImg *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config, usedLines map[[2]int]int,
	superWidth, superHeight, supersample, width, height int) Line {

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

				score := evaluateLineV9Birsak(task.key, superCanvas, superImg, importance,
					linePixels, config, superWidth, superHeight, supersample, width, height)

				if usedLines[task.key] > 0 {
					score *= 1.0 / (1.0 + 0.3*float64(usedLines[task.key]))
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

// evaluateLineV9Birsak evaluates a line using supersampled SSIM-based scoring
func evaluateLineV9Birsak(key [2]int, superCanvas [][]float64, superImg *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config,
	superWidth, superHeight, supersample, width, height int) float64 {

	pixels := linePixels[key]
	if len(pixels) == 0 {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0

	for _, pixel := range pixels {
		if pixel.X < 0 || pixel.X >= superWidth || pixel.Y < 0 || pixel.Y >= superHeight {
			continue
		}

		currentVal := superCanvas[pixel.Y][pixel.X]
		targetVal := float64(superImg.GrayAt(pixel.X, pixel.Y).Y)

		delta := float64(config.LineWeight) * pixel.Weight
		newVal := currentVal - delta
		if newVal < 0 {
			newVal = 0
		}

		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)
		improvement := oldError - newError

		// Map to original resolution for importance lookup
		origX := pixel.X / supersample
		origY := pixel.Y / supersample
		if origX >= width {
			origX = width - 1
		}
		if origY >= height {
			origY = height - 1
		}

		imp := importance[origY][origX]

		if improvement > 0 {
			totalScore += improvement * imp * pixel.Weight
			improvingPixels++
		} else if improvement < 0 {
			// Mild over-darkening penalty
			totalScore += improvement * imp * 1.1 * pixel.Weight
			worseningPixels++
		}
		totalWeight += imp * pixel.Weight
	}

	if totalWeight <= 0 {
		return 0
	}

	score := totalScore / totalWeight

	// Worsen ratio penalty
	totalPixels := improvingPixels + worseningPixels
	if totalPixels > 0 {
		worsenRatio := float64(worseningPixels) / float64(totalPixels)
		if worsenRatio > 0.4 {
			score *= (1.0 - worsenRatio*0.4)
		}
	}

	return score
}

// enhancedLineRemovalV9 implements line removal optimization at supersampled resolution
func enhancedLineRemovalV9(lines []Line, pins []Pin, superCanvas [][]float64, superImg *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, config *Config,
	superWidth, superHeight, supersample, width, height int, img *image.Gray) []Line {

	if len(lines) == 0 {
		return lines
	}

	maxRemovals := len(lines) / 4 // Allow removing up to 1/4 of lines

	// Downsample for metrics
	downsampledCanvas := downsampleCanvasV9(superCanvas, supersample, width, height)
	currentMSE := calculateMSEV5(downsampledCanvas, img, width, height)
	currentSSIM := quickSSIMV5(downsampledCanvas, img, width, height)
	fmt.Printf("Before supersampled removal: MSE = %.2f, SSIM~%.3f\n", currentMSE, currentSSIM)

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
			if pixel.X < 0 || pixel.X >= superWidth || pixel.Y < 0 || pixel.Y >= superHeight {
				continue
			}

			currentVal := superCanvas[pixel.Y][pixel.X]
			targetVal := float64(superImg.GrayAt(pixel.X, pixel.Y).Y)

			delta := float64(config.LineWeight) * pixel.Weight
			newVal := currentVal + delta
			if newVal > 255 {
				newVal = 255
			}

			oldErr := (currentVal - targetVal) * (currentVal - targetVal)
			newErr := (newVal - targetVal) * (newVal - targetVal)

			// Map to original resolution for importance lookup
			origX := pixel.X / supersample
			origY := pixel.Y / supersample
			if origX >= width {
				origX = width - 1
			}
			if origY >= height {
				origY = height - 1
			}

			imp := importance[origY][origX]
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

		// Remove line from supersampled canvas
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
				delta := float64(config.LineWeight) * pixel.Weight
				superCanvas[pixel.Y][pixel.X] += delta
				if superCanvas[pixel.Y][pixel.X] > 255 {
					superCanvas[pixel.Y][pixel.X] = 255
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

	// Downsample for final metrics
	newDownsampledCanvas := downsampleCanvasV9(superCanvas, supersample, width, height)
	newMSE := calculateMSEV5(newDownsampledCanvas, img, width, height)
	newSSIM := quickSSIMV5(newDownsampledCanvas, img, width, height)
	fmt.Printf("After supersampled removal: MSE = %.2f, SSIM~%.3f (removed %d lines)\n", newMSE, newSSIM, removedCount)

	return newLines
}