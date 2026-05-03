package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV12Focused implements focused improvements for better SSIM:
// 1. Enhanced importance mapping with better face detection
// 2. Improved add/remove optimization 
// 3. Better source-over alpha calibration
// 4. SSIM-aware scoring without supersampling overhead
func GenerateStringArtV12Focused(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v12.0 Focused ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	// Create enhanced importance map with better face detection
	importance := createImportanceMapV12(img, edgeMap, width, height)

	// Pre-compute line pixels with enhanced anti-aliasing
	fmt.Println("Pre-computing line pixels with enhanced anti-aliasing...")
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

	// Phase 1: Enhanced Greedy Add Phase with better scoring
	fmt.Println("\n--- Phase 1: Enhanced Greedy Add Phase ---")
	recentScores := make([]float64, 0, 30)
	stagnationCount := 0
	baseWeight := float64(config.LineWeight)

	for i := 0; i < config.NumLines; i++ {
		// Improved adaptive line weight with better calibration
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.4*progress)
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV12(currentPin, pins, canvas, img, importance,
			linePixels, config, usedLines, width, height, adaptiveWeight)

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
			if avgScore < 0.3 {
				stagnationCount++
				if stagnationCount > 20 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Apply line to canvas with improved source-over compositing
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				// Improved source-over compositing with better alpha calibration
				alpha := (adaptiveWeight / 255.0) * 0.8 // Calibrated for better mobile rendering match
				canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[key]++

		if (i+1)%200 == 0 {
			mse := calculateMSEV12(canvas, img, width, height)
			ssim := calculateSSIMApproxV12(canvas, img, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM~%.3f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, adaptiveWeight)
		}
	}

	// Phase 2: Enhanced Add/Remove Optimization with better evaluation
	fmt.Printf("\n--- Phase 2: Enhanced Add/Remove Optimization ---\n")
	fmt.Printf("Starting optimization with %d lines\n", len(lines))

	// Multiple optimization passes with improved evaluation
	for pass := 1; pass <= 2; pass++ {
		fmt.Printf("Optimization pass %d/2\n", pass)
		
		beforeMSE := calculateMSEV12(canvas, img, width, height)
		beforeSSIM := calculateSSIMApproxV12(canvas, img, width, height)
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
				if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
					// Reverse source-over compositing
					alpha := (baseWeight / 255.0) * 0.8
					if 1.0-alpha*pixel.Weight > 0.001 {
						canvas[pixel.Y][pixel.X] = (canvas[pixel.Y][pixel.X] - 0.0*alpha*pixel.Weight) / (1.0-alpha*pixel.Weight)
					}
				}
			}
			
			// Evaluate quality without this line
			testSSIM := calculateSSIMApproxV12(canvas, img, width, height)
			
			if testSSIM > beforeSSIM + 0.002 {
				// Removing this line improves quality - keep it removed
				lines = append(lines[:i], lines[i+1:]...)
				removedCount++
				beforeSSIM = testSSIM
			} else {
				// Removing this line hurts quality - put it back
				for _, pixel := range pixels {
					if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
						alpha := (baseWeight / 255.0) * 0.8
						canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
					}
				}
			}
		}
		
		afterMSE := calculateMSEV12(canvas, img, width, height)
		afterSSIM := calculateSSIMApproxV12(canvas, img, width, height)
		fmt.Printf("After pass %d: MSE = %.2f, SSIM~%.3f (removed %d lines)\n", pass, afterMSE, afterSSIM, removedCount)
		
		if removedCount == 0 {
			fmt.Printf("No lines removed in pass %d, stopping optimization\n", pass)
			break
		}
	}

	// Final evaluation
	finalMSE := calculateMSEV12(canvas, img, width, height)
	finalSSIM := calculateSSIMApproxV12(canvas, img, width, height)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	return lines, canvas
}

// Enhanced importance map with better face detection
func createImportanceMapV12(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	centerX, centerY := float64(width)/2, float64(height)/2

	// Enhanced face region detection (more precise)
	faceRegionTop := int(float64(height) * 0.20)
	faceRegionBottom := int(float64(height) * 0.70)
	faceRegionLeft := int(float64(width) * 0.30)
	faceRegionRight := int(float64(width) * 0.70)

	// Eye regions (more specific)
	leftEyeX := int(float64(width) * 0.40)
	rightEyeX := int(float64(width) * 0.60)
	eyeY := int(float64(height) * 0.35)
	eyeRadius := int(float64(width) * 0.08)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Base importance from edge strength
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + edgeStrength*3.0

			// Distance-based weighting (center is more important)
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			distance := math.Sqrt(dx*dx + dy*dy)
			maxDistance := math.Sqrt(centerX*centerX + centerY*centerY)
			distanceWeight := 1.0 + (1.0-distance/maxDistance)*0.8

			// Face region boost
			faceWeight := 1.0
			if x >= faceRegionLeft && x <= faceRegionRight && y >= faceRegionTop && y <= faceRegionBottom {
				faceWeight = 2.0 // Moderate boost for face region
			}

			// Eye region boost (very specific)
			eyeWeight := 1.0
			leftEyeDist := math.Sqrt(float64((x-leftEyeX)*(x-leftEyeX) + (y-eyeY)*(y-eyeY)))
			rightEyeDist := math.Sqrt(float64((x-rightEyeX)*(x-rightEyeX) + (y-eyeY)*(y-eyeY)))
			if leftEyeDist < float64(eyeRadius) || rightEyeDist < float64(eyeRadius) {
				eyeWeight = 3.0 // Strong boost for eye regions
			}

			// High contrast areas (dark features)
			grayValue := float64(img.GrayAt(x, y).Y)
			contrastWeight := 1.0
			if grayValue < 100 {
				contrastWeight = 2.2 // Strong boost for dark areas (eyes, nose, mouth)
			} else if grayValue > 180 {
				contrastWeight = 1.3 // Moderate boost for bright areas
			}

			importance[y][x] *= distanceWeight * faceWeight * eyeWeight * contrastWeight
		}
	}

	return importance
}

// Enhanced line finding with better scoring
func findBestLineV12(currentPin int, pins []Pin, canvas [][]float64, target *image.Gray, importance [][]float64,
	linePixels map[[2]int][]AntiAliasedPixel, config *Config, usedLines map[[2]int]int,
	width, height int, lineWeight float64) Line {

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
		if usedLines[key] >= 2 { // Limit reuse more strictly
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

				// Calculate improved score
				score := calculateLineScoreV12(pixels, canvas, target, importance, width, height, lineWeight)

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

// Improved line scoring with better perceptual weighting
func calculateLineScoreV12(pixels []AntiAliasedPixel, canvas [][]float64, target *image.Gray, importance [][]float64,
	width, height int, lineWeight float64) float64 {

	// Calculate improvement with better weighting
	totalImprovement := 0.0
	totalImportance := 0.0
	alpha := (lineWeight / 255.0) * 0.8

	for _, pixel := range pixels {
		if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
			targetVal := float64(target.GrayAt(pixel.X, pixel.Y).Y)
			beforeVal := canvas[pixel.Y][pixel.X]
			afterVal := beforeVal*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			
			beforeError := (beforeVal - targetVal) * (beforeVal - targetVal)
			afterError := (afterVal - targetVal) * (afterVal - targetVal)
			improvement := beforeError - afterError
			
			pixelImportance := importance[pixel.Y][pixel.X]
			totalImprovement += improvement * pixelImportance * pixel.Weight
			totalImportance += pixelImportance * pixel.Weight
		}
	}

	if totalImportance > 0 {
		return totalImprovement / totalImportance
	}
	return 0
}

// Improved MSE calculation
func calculateMSEV12(canvas [][]float64, img *image.Gray, width, height int) float64 {
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

// Improved SSIM approximation
func calculateSSIMApproxV12(canvas [][]float64, target *image.Gray, width, height int) float64 {
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