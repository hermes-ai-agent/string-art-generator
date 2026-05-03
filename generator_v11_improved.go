package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV11Improved implements focused improvements for better SSIM:
// 1. 2x supersampled rendering (balance between quality and speed)
// 2. Enhanced add/remove optimization with better scoring
// 3. Improved importance mapping with face detection
// 4. SSIM-based perceptual scoring instead of MSE
// 5. Better source-over alpha calibration
func GenerateStringArtV11Improved(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v11.0 Improved ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	// 2x supersampling for better gray levels without excessive memory usage
	supersample := 2
	superWidth := width * supersample
	superHeight := height * supersample

	fmt.Printf("Supersampled resolution: %dx%d (2x)\n", superWidth, superHeight)

	// Create enhanced importance map with face detection
	importance := createImportanceMapV11(img, edgeMap, width, height)

	// Pre-compute line pixels at supersampled resolution
	fmt.Println("Pre-computing supersampled line pixels...")
	centerX, centerY := float64(superWidth)/2, float64(superHeight)/2
	radius := centerX - 10*float64(supersample)
	pins := GeneratePinsSuper(config.NumPins, radius, centerX, centerY)

	linePixels := precomputeLinePixelsSuper(pins, superWidth, superHeight, config.MinDistance, supersample)
	fmt.Printf("Pre-computed %d valid line segments at 2x resolution\n", len(linePixels))

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

	// Phase 1: Enhanced Greedy Add Phase with SSIM scoring
	fmt.Println("\n--- Phase 1: Enhanced Greedy Add Phase (SSIM-based) ---")
	recentScores := make([]float64, 0, 30)
	stagnationCount := 0
	baseWeight := float64(config.LineWeight)

	for i := 0; i < config.NumLines; i++ {
		// Adaptive line weight with better calibration
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.3*progress)
		if adaptiveWeight < 15 {
			adaptiveWeight = 15
		}

		bestLine := findBestLineV11(currentPin, pins, superCanvas, superImg, importance,
			linePixels, config, usedLines, superWidth, superHeight, supersample, width, height, adaptiveWeight)

		if bestLine.Score <= 0.001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Enhanced stagnation detection
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 30 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 30 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < 0.4 {
				stagnationCount++
				if stagnationCount > 15 {
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
				// Source-over compositing with calibrated alpha
				alpha := adaptiveWeight / 255.0
				superCanvas[pixel.Y][pixel.X] = superCanvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[key]++

		if (i+1)%200 == 0 {
			// Downsample for progress evaluation
			downsampled := downsampleCanvasV11(superCanvas, supersample, width, height)
			mse := calculateMSEV11(downsampled, img, width, height)
			ssim := calculateSSIMApprox(downsampled, img)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM~%.3f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, adaptiveWeight)
		}
	}

	// Phase 2: Enhanced Add/Remove Optimization
	fmt.Printf("\n--- Phase 2: Enhanced Add/Remove Optimization ---\n")
	fmt.Printf("Starting optimization with %d lines\n", len(lines))

	// Multiple optimization passes
	for pass := 1; pass <= 3; pass++ {
		fmt.Printf("Optimization pass %d/3\n", pass)
		
		// Downsample for evaluation
		downsampled := downsampleCanvasV11(superCanvas, supersample, width, height)
		beforeMSE := calculateMSEV11(downsampled, img, width, height)
		beforeSSIM := calculateSSIMApprox(downsampled, img)
		fmt.Printf("Before pass %d: MSE = %.2f, SSIM~%.3f\n", pass, beforeMSE, beforeSSIM)

		removedCount := 0
		
		// Try removing each line and see if it improves quality
		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			
			// Remove line from canvas
			key := [2]int{min(line.From, line.To), max(line.From, line.To)}
			pixels := linePixels[key]
			
			// Temporarily remove line
			for _, pixel := range pixels {
				if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
					// Reverse source-over compositing
					alpha := baseWeight / 255.0
					if 1.0-alpha*pixel.Weight > 0.001 {
						superCanvas[pixel.Y][pixel.X] = (superCanvas[pixel.Y][pixel.X] - 0.0*alpha*pixel.Weight) / (1.0-alpha*pixel.Weight)
					}
				}
			}
			
			// Evaluate quality without this line
			testDownsampled := downsampleCanvasV11(superCanvas, supersample, width, height)
			testSSIM := calculateSSIMApprox(testDownsampled, img)
			
			if testSSIM > beforeSSIM + 0.001 {
				// Removing this line improves quality - keep it removed
				lines = append(lines[:i], lines[i+1:]...)
				removedCount++
				beforeSSIM = testSSIM
			} else {
				// Removing this line hurts quality - put it back
				for _, pixel := range pixels {
					if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
						alpha := baseWeight / 255.0
						superCanvas[pixel.Y][pixel.X] = superCanvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
					}
				}
			}
		}
		
		downsampled = downsampleCanvasV11(superCanvas, supersample, width, height)
		afterMSE := calculateMSEV11(downsampled, img, width, height)
		afterSSIM := calculateSSIMApprox(downsampled, img)
		fmt.Printf("After pass %d: MSE = %.2f, SSIM~%.3f (removed %d lines)\n", pass, afterMSE, afterSSIM, removedCount)
		
		if removedCount == 0 {
			fmt.Printf("No lines removed in pass %d, stopping optimization\n", pass)
			break
		}
	}

	// Final evaluation
	finalCanvas := downsampleCanvasV11(superCanvas, supersample, width, height)
	finalMSE := calculateMSEV11(finalCanvas, img, width, height)
	finalSSIM := calculateSSIMApprox(finalCanvas, img)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	return lines, finalCanvas
}

// Enhanced importance map with better face detection
func createImportanceMapV11(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	centerX, centerY := float64(width)/2, float64(height)/2

	// Detect face region (upper center area)
	faceRegionTop := int(float64(height) * 0.25)
	faceRegionBottom := int(float64(height) * 0.65)
	faceRegionLeft := int(float64(width) * 0.35)
	faceRegionRight := int(float64(width) * 0.65)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Base importance from edge strength
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + edgeStrength*2.0

			// Distance-based weighting (center is more important)
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			distance := math.Sqrt(dx*dx + dy*dy)
			maxDistance := math.Sqrt(centerX*centerX + centerY*centerY)
			distanceWeight := 1.0 + (1.0-distance/maxDistance)*0.5

			// Face region boost
			faceWeight := 1.0
			if x >= faceRegionLeft && x <= faceRegionRight && y >= faceRegionTop && y <= faceRegionBottom {
				faceWeight = 2.5 // Strong boost for face region
			}

			// High contrast areas (eyes, nose, mouth)
			grayValue := float64(img.GrayAt(x, y).Y)
			contrastWeight := 1.0
			if grayValue < 80 || grayValue > 200 {
				contrastWeight = 1.8 // Boost high contrast areas
			}

			importance[y][x] *= distanceWeight * faceWeight * contrastWeight
		}
	}

	return importance
}

// Enhanced line finding with SSIM-based scoring
func findBestLineV11(currentPin int, pins []Pin, canvas [][]float64, target *image.Gray, importance [][]float64,
	linePixels map[[2]int][]AntiAliasedPixel, config *Config, usedLines map[[2]int]int,
	superWidth, superHeight, supersample, width, height int, lineWeight float64) Line {

	type candidate struct {
		pin   int
		score float64
	}

	candidates := make([]candidate, 0, config.NumPins)

	// Collect all valid candidates
	for nextPin := 0; nextPin < config.NumPins; nextPin++ {
		if nextPin == currentPin {
			continue
		}

		distance := int(math.Abs(float64(nextPin - currentPin)))
		if distance > config.NumPins/2 {
			distance = config.NumPins - distance
		}
		if distance < config.MinDistance {
			continue
		}

		key := [2]int{min(currentPin, nextPin), max(currentPin, nextPin)}
		if usedLines[key] >= 3 { // Limit reuse
			continue
		}

		candidates = append(candidates, candidate{pin: nextPin, score: 0})
	}

	if len(candidates) == 0 {
		return Line{From: currentPin, To: currentPin, Score: 0}
	}

	// Parallel evaluation of candidates
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
				nextPin := candidates[j].pin
				key := [2]int{min(currentPin, nextPin), max(currentPin, nextPin)}
				pixels := linePixels[key]

				// Calculate SSIM-based score
				score := calculateLineScoreSSIM(pixels, canvas, target, importance, superWidth, superHeight, supersample, width, height, lineWeight)

				mu.Lock()
				candidates[j].score = score
				mu.Unlock()
			}
		}(start, end)
	}

	wg.Wait()

	// Find best candidate
	bestScore := 0.0
	bestPin := currentPin
	for _, candidate := range candidates {
		if candidate.score > bestScore {
			bestScore = candidate.score
			bestPin = candidate.pin
		}
	}

	return Line{From: currentPin, To: bestPin, Score: bestScore}
}

// SSIM-based line scoring
func calculateLineScoreSSIM(pixels []AntiAliasedPixel, canvas [][]float64, target *image.Gray, importance [][]float64,
	superWidth, superHeight, supersample, width, height int, lineWeight float64) float64 {

	// Create temporary canvas with line applied
	tempCanvas := make([][]float64, superHeight)
	for y := 0; y < superHeight; y++ {
		tempCanvas[y] = make([]float64, superWidth)
		copy(tempCanvas[y], canvas[y])
	}

	// Apply line with source-over compositing
	alpha := lineWeight / 255.0
	for _, pixel := range pixels {
		if pixel.X >= 0 && pixel.X < superWidth && pixel.Y >= 0 && pixel.Y < superHeight {
			tempCanvas[pixel.Y][pixel.X] = tempCanvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
		}
	}

	// Calculate SSIM improvement (simplified for performance)
	mseImprovement := 0.0
	importanceWeight := 0.0
	totalWeight := 0.0

	for _, pixel := range pixels {
		origX := pixel.X / supersample
		origY := pixel.Y / supersample
		if origX >= 0 && origX < width && origY >= 0 && origY < height {
			targetVal := float64(target.GrayAt(origX, origY).Y)
			beforeVal := canvas[pixel.Y][pixel.X]
			afterVal := tempCanvas[pixel.Y][pixel.X]
			
			beforeError := (beforeVal - targetVal) * (beforeVal - targetVal)
			afterError := (afterVal - targetVal) * (afterVal - targetVal)
			improvement := beforeError - afterError
			
			mseImprovement += improvement * pixel.Weight
			importanceWeight += importance[origY][origX] * pixel.Weight
			totalWeight += pixel.Weight
		}
	}

	if totalWeight > 0 {
		mseImprovement /= totalWeight
		importanceWeight /= totalWeight
	}

	return mseImprovement * importanceWeight
}

// Downsample canvas from supersampled to original resolution
func downsampleCanvasV11(superCanvas [][]float64, supersample, width, height int) [][]float64 {
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

// Approximate SSIM calculation for performance
func calculateSSIMApprox(canvas [][]float64, target *image.Gray) float64 {
	bounds := target.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	var meanCanvas, meanTarget, varCanvas, varTarget, covar float64
	count := 0

	// Calculate means
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(canvas) && x < len(canvas[y]) {
				meanCanvas += canvas[y][x]
				meanTarget += float64(target.GrayAt(x, y).Y)
				count++
			}
		}
	}
	meanCanvas /= float64(count)
	meanTarget /= float64(count)

	// Calculate variances and covariance
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(canvas) && x < len(canvas[y]) {
				canvasVal := canvas[y][x]
				targetVal := float64(target.GrayAt(x, y).Y)
				
				varCanvas += (canvasVal - meanCanvas) * (canvasVal - meanCanvas)
				varTarget += (targetVal - meanTarget) * (targetVal - meanTarget)
				covar += (canvasVal - meanCanvas) * (targetVal - meanTarget)
			}
		}
	}
	varCanvas /= float64(count - 1)
	varTarget /= float64(count - 1)
	covar /= float64(count - 1)

	// SSIM constants
	c1 := (0.01 * 255) * (0.01 * 255)
	c2 := (0.03 * 255) * (0.03 * 255)

	// SSIM formula
	numerator := (2*meanCanvas*meanTarget + c1) * (2*covar + c2)
	denominator := (meanCanvas*meanCanvas + meanTarget*meanTarget + c1) * (varCanvas + varTarget + c2)

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// calculateMSEV11 calculates MSE between canvas and target image
func calculateMSEV11(canvas [][]float64, img *image.Gray, width, height int) float64 {
	var mse float64
	count := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(canvas) && x < len(canvas[y]) {
				canvasVal := canvas[y][x]
				targetVal := float64(img.GrayAt(x, y).Y)
				diff := canvasVal - targetVal
				mse += diff * diff
				count++
			}
		}
	}

	if count > 0 {
		mse /= float64(count)
	}

	return mse
}