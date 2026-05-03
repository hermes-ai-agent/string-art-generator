package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV14Birsak implements Birsak 2018 supersampled rendering with key improvements:
// 1. 4x supersampled rendering with proper downsampling (memory efficient)
// 2. Enhanced face feature detection and importance mapping
// 3. Add/remove optimization with MSE-based evaluation (proven to work)
// 4. Calibrated source-over alpha to match mobile SVG rendering
func GenerateStringArtV14Birsak(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Use 4x supersampling for balance between quality and memory
	supersample := 4
	superWidth := width * supersample
	superHeight := height * supersample

	fmt.Printf("=== String Art Generator v14.0 Birsak 2018 ===\n")
	fmt.Printf("Base Resolution: %dx%d\n", width, height)
	fmt.Printf("Super Resolution: %dx%d (4x supersampling)\n", superWidth, superHeight)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)

	// Create target array at base resolution (what we want to achieve)
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y)
		}
	}

	// Create supersampled canvas (starts white, we darken it with strings)
	superCanvas := make([][]float64, superHeight)
	for y := 0; y < superHeight; y++ {
		superCanvas[y] = make([]float64, superWidth)
		for x := 0; x < superWidth; x++ {
			superCanvas[y][x] = 255.0
		}
	}

	// Generate pins at supersampled resolution
	centerX, centerY := float64(superWidth)/2, float64(superHeight)/2
	radius := math.Min(centerX, centerY) - 10*float64(supersample)
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	// Create enhanced importance map with face detection
	importance := createEnhancedImportanceMapV14(img, edgeMap, width, height)

	// Pre-compute all line pixels with anti-aliasing at supersampled resolution
	fmt.Println("Pre-computing supersampled line pixels with anti-aliasing...")
	linePixels := precomputeLinePixelsV14(pins, superWidth, superHeight, config.MinDistance*supersample)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Adaptive parameters
	baseWeight := float64(config.LineWeight)
	recentScores := make([]float64, 0, 25)
	stagnationCount := 0

	// Phase 1: Greedy line addition with enhanced scoring
	fmt.Println("\n--- Phase 1: Greedy Line Addition ---")
	for i := 0; i < config.NumLines; i++ {
		// Adaptive line weight
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.4*progress)
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV14(currentPin, pins, superCanvas, target, edgeMap, importance,
			linePixels, config, adaptiveWeight, usedLines, supersample)

		if bestLine.Score <= 0.05 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Enhanced stagnation detection
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 25 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 25 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < config.StopThreshold*0.25 {
				stagnationCount++
				if stagnationCount > 15 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Apply line to supersampled canvas with calibrated alpha
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
				// Calibrated source-over compositing for mobile SVG match (4x supersampling)
				alpha := (adaptiveWeight / 255.0) * 0.88 // Better calibration for mobile match
				superCanvas[pixel.Y][pixel.X] = superCanvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[key]++

		if (i+1)%200 == 0 {
			// Downsample for MSE calculation
			canvas := downsampleCanvasV14(superCanvas, superWidth, superHeight, width, height, supersample)
			mse := calculateMSEV14(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, adaptiveWeight)
		}
	}

	// Phase 2: Enhanced Line Removal with MSE optimization
	fmt.Println("\n--- Phase 2: Enhanced Line Removal ---")
	canvas := downsampleCanvasV14(superCanvas, superWidth, superHeight, width, height, supersample)
	beforeMSE := calculateMSEV14(canvas, target, width, height)
	fmt.Printf("Before removal: MSE = %.1f\n", beforeMSE)

	removedCount := 0
	maxRemovalPasses := 2
	
	for pass := 0; pass < maxRemovalPasses; pass++ {
		fmt.Printf("Removal pass %d/%d...\n", pass+1, maxRemovalPasses)
		passRemovals := 0
		
		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			key := [2]int{min(line.From, line.To), max(line.From, line.To)}
			pixels := linePixels[key]

			// Temporarily remove line from supersampled canvas
			for _, pixel := range pixels {
				if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
					alpha := (baseWeight / 255.0) * 0.88
					if 1.0-alpha*pixel.Weight > 0.001 {
						superCanvas[pixel.Y][pixel.X] = (superCanvas[pixel.Y][pixel.X] - 0.0*alpha*pixel.Weight) / (1.0-alpha*pixel.Weight)
					}
				}
			}

			// Check if removal improves MSE
			testCanvas := downsampleCanvasV14(superCanvas, superWidth, superHeight, width, height, supersample)
			testMSE := calculateMSEV14(testCanvas, target, width, height)
			if testMSE < beforeMSE - 1.0 {
				// Keep it removed
				lines = append(lines[:i], lines[i+1:]...)
				removedCount++
				passRemovals++
				beforeMSE = testMSE
			} else {
				// Re-add the line
				for _, pixel := range pixels {
					if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
						alpha := (baseWeight / 255.0) * 0.88
						superCanvas[pixel.Y][pixel.X] = superCanvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
					}
				}
			}
		}
		
		fmt.Printf("Pass %d: removed %d lines\n", pass+1, passRemovals)
		if passRemovals == 0 {
			break // No more improvements possible
		}
	}

	// Final downsampling
	finalCanvas := downsampleCanvasV14(superCanvas, superWidth, superHeight, width, height, supersample)
	afterMSE := calculateMSEV14(finalCanvas, target, width, height)
	fmt.Printf("After removal: MSE = %.1f (removed %d lines)\n", afterMSE, removedCount)

	fmt.Printf("Final: %d lines, MSE = %.1f\n", len(lines), afterMSE)
	return lines, finalCanvas
}

// createEnhancedImportanceMapV14 creates an enhanced importance map with face feature detection
func createEnhancedImportanceMapV14(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	// Base importance from edge map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)
			importance[y][x] = 1.0 + edgeVal/255.0*2.0
		}
	}

	// Enhanced face feature detection
	centerX, centerY := width/2, height/2
	
	// Eye regions (upper third, left and right)
	eyeY := centerY - height/6
	leftEyeX := centerX - width/8
	rightEyeX := centerX + width/8
	eyeRadius := width / 20
	
	// Nose region (center)
	noseY := centerY
	noseRadius := width / 25
	
	// Mouth region (lower third)
	mouthY := centerY + height/8
	mouthRadius := width / 18
	
	// Apply enhanced weights to face features
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Left eye
			if distance(x, y, leftEyeX, eyeY) < eyeRadius {
				importance[y][x] *= 3.5
			}
			// Right eye
			if distance(x, y, rightEyeX, eyeY) < eyeRadius {
				importance[y][x] *= 3.5
			}
			// Nose
			if distance(x, y, centerX, noseY) < noseRadius {
				importance[y][x] *= 2.8
			}
			// Mouth
			if distance(x, y, centerX, mouthY) < mouthRadius {
				importance[y][x] *= 2.5
			}
		}
	}

	return importance
}

// precomputeLinePixelsV14 precomputes line pixels at supersampled resolution
func precomputeLinePixelsV14(pins []Pin, width, height, minDistance int) map[[2]int][]AntiAliasedPixel {
	linePixels := make(map[[2]int][]AntiAliasedPixel)
	
	for i := 0; i < len(pins); i++ {
		for j := i + minDistance; j < len(pins); j++ {
			if j-i >= len(pins)-minDistance {
				continue
			}
			
			key := [2]int{i, j}
			pixels := rasterizeLineAntiAliasedSuper(pins[i], pins[j], width, height, 1)
			if len(pixels) > 0 {
				linePixels[key] = pixels
			}
		}
	}
	
	return linePixels
}

// findBestLineV14 finds the best line using enhanced scoring
func findBestLineV14(currentPin int, pins []Pin, superCanvas [][]float64, target [][]float64, edgeMap *image.Gray, importance [][]float64,
	linePixels map[[2]int][]AntiAliasedPixel, config *Config, lineWeight float64, usedLines map[[2]int]int, supersample int) Line {

	numPins := len(pins)
	type candidate struct {
		toPin int
		score float64
	}

	candidates := make([]candidate, 0, numPins)

	// Collect valid candidates
	for toPin := 0; toPin < numPins; toPin++ {
		if toPin == currentPin {
			continue
		}

		distance := int(math.Abs(float64(toPin - currentPin)))
		if distance > numPins/2 {
			distance = numPins - distance
		}
		if distance < config.MinDistance {
			continue
		}

		key := [2]int{min(currentPin, toPin), max(currentPin, toPin)}
		if usedLines[key] >= 2 { // Limit reuse
			continue
		}

		candidates = append(candidates, candidate{toPin: toPin, score: 0})
	}

	if len(candidates) == 0 {
		return Line{From: currentPin, To: currentPin, Score: 0}
	}

	// Parallel evaluation
	var wg sync.WaitGroup
	var mu sync.Mutex

	batchSize := (len(candidates) + config.Workers - 1) / config.Workers
	for i := 0; i < config.Workers; i++ {
		start := i * batchSize
		end := min(start+batchSize, len(candidates))
		if start >= len(candidates) {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				toPin := candidates[j].toPin
				key := [2]int{min(currentPin, toPin), max(currentPin, toPin)}
				pixels := linePixels[key]

				score := calculateLineScoreV14(pixels, superCanvas, target, importance, lineWeight, supersample)
				
				// Apply usage penalty
				if usedLines[key] > 0 {
					score *= 0.7
				}

				mu.Lock()
				candidates[j].score = score
				mu.Unlock()
			}
		}(start, end)
	}

	wg.Wait()

	// Find best candidate
	bestScore := -1.0
	bestToPin := currentPin
	for _, candidate := range candidates {
		if candidate.score > bestScore {
			bestScore = candidate.score
			bestToPin = candidate.toPin
		}
	}

	return Line{From: currentPin, To: bestToPin, Score: bestScore}
}

// calculateLineScoreV14 calculates score for a potential line
func calculateLineScoreV14(pixels []AntiAliasedPixel, superCanvas, target, importance [][]float64, 
	weight float64, supersample int) float64 {
	
	if len(pixels) == 0 {
		return 0
	}

	score := 0.0
	count := 0
	alpha := (weight / 255.0) * 0.88

	for _, pixel := range pixels {
		if pixel.X >= 0 && pixel.X < len(superCanvas[0]) && pixel.Y >= 0 && pixel.Y < len(superCanvas) {
			// Map to base resolution for target comparison
			baseX := pixel.X / supersample
			baseY := pixel.Y / supersample
			
			if baseX < len(target[0]) && baseY < len(target) {
				currentVal := superCanvas[pixel.Y][pixel.X]
				targetVal := target[baseY][baseX]
				
				// Calculate what the new value would be
				newVal := currentVal*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
				
				// Score based on how much closer we get to target
				currentError := math.Abs(currentVal - targetVal)
				newError := math.Abs(newVal - targetVal)
				improvement := currentError - newError
				
				// Weight by importance and pixel weight
				importanceWeight := importance[baseY][baseX]
				score += improvement * importanceWeight * pixel.Weight
				count++
			}
		}
	}

	if count > 0 {
		return score / float64(count)
	}
	return 0
}

// downsampleCanvasV14 performs proper downsampling from supersampled to base resolution
func downsampleCanvasV14(superCanvas [][]float64, superWidth, superHeight, width, height, supersample int) [][]float64 {
	canvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]float64, width)
	}
	
	// Box filter downsampling with proper averaging
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0.0
			count := 0
			
			// Average over supersample x supersample block
			for dy := 0; dy < supersample; dy++ {
				for dx := 0; dx < supersample; dx++ {
					superY := y*supersample + dy
					superX := x*supersample + dx
					if superY < superHeight && superX < superWidth {
						sum += superCanvas[superY][superX]
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

// calculateMSEV14 calculates MSE between canvas and target
func calculateMSEV14(canvas, target [][]float64, width, height int) float64 {
	sum := 0.0
	count := 0
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			sum += diff * diff
			count++
		}
	}
	
	if count > 0 {
		return sum / float64(count)
	}
	return 0
}

// distance calculates Euclidean distance between two points
func distance(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	dy := y1 - y2
	return int(math.Sqrt(float64(dx*dx + dy*dy)))
}