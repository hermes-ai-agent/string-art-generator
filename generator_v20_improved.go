package main

import (
	"fmt"
	"image"
	"math"
)

// GenerateStringArtV20Improved implements improvements over v5.0 for v3.3.0+ requirements:
// 1. Enhanced importance mapping with better face detection
// 2. Improved add/remove optimization with SSIM-based scoring  
// 3. Better line weight adaptation
// 4. Perceptual scoring improvements
func GenerateStringArtV20Improved(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v20.0 Improved ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)

	// Create target array (what we want to achieve)
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y)
		}
	}

	// Create canvas (starts white, we darken it with strings)
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

	// Create enhanced importance map with better face detection
	importance := createV20ImportanceMap(img, edgeMap, width, height)

	// Pre-compute all line pixels with anti-aliasing
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Enhanced adaptive parameters
	baseWeight := float64(config.LineWeight)
	recentScores := make([]float64, 0, 50)

	// Phase 1: Greedy line addition with enhanced scoring
	fmt.Println("\n--- Phase 1: Enhanced Greedy Line Addition ---")
	for i := 0; i < config.NumLines; i++ {
		// Enhanced adaptive line weight - more gradual reduction
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.35*progress)
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV20(currentPin, pins, canvas, target, importance,
			linePixels, adaptiveWeight, usedLines)

		if bestLine.Score <= 50.0 { // Lower threshold for better continuation
			fmt.Printf("Stopping at line %d (no improvement possible, score: %.2f)\n", i, bestLine.Score)
			break
		}

		// Enhanced stagnation detection
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 50 {
			recentScores = recentScores[1:]
		}

		// Apply the line to canvas
		applyLineToCanvasV20(canvas, linePixels[[2]int{bestLine.From, bestLine.To}], adaptiveWeight)

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[[2]int{bestLine.From, bestLine.To}]++

		if (i+1)%200 == 0 {
			// Compute current metrics for progress tracking
			currentSSIM := computeCanvasSSIMV20(canvas, target)
			currentMSE := computeWeightedMSEV20(canvas, target, importance)
			avgRecentScore := 0.0
			if len(recentScores) > 0 {
				for _, s := range recentScores {
					avgRecentScore += s
				}
				avgRecentScore /= float64(len(recentScores))
			}
			fmt.Printf("Progress: %d/%d lines (SSIM: %.4f, MSE: %.1f, score: %.1f, weight: %.1f)\n", 
				i+1, config.NumLines, currentSSIM, currentMSE, avgRecentScore, adaptiveWeight)
		}
	}

	fmt.Printf("Phase 1 complete: %d lines added\n", len(lines))

	// Phase 2: Enhanced SSIM-based line removal
	fmt.Println("\n--- Phase 2: SSIM-based Line Removal ---")
	initialSSIM := computeCanvasSSIMV20(canvas, target)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	removed := 0
	maxRemovals := 150
	improvementThreshold := 0.0003

	// Multiple passes of removal
	for pass := 0; pass < 3 && removed < maxRemovals; pass++ {
		fmt.Printf("Removal pass %d...\n", pass+1)
		passRemoved := 0
		
		// Test removing lines (focus on recent lines first)
		for i := len(lines) - 1; i >= 0 && removed < maxRemovals; i-- {
			line := lines[i]
			
			// Remove line temporarily
			removeLineFromCanvasV20(canvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight))
			
			// Check if SSIM improved
			newSSIM := computeCanvasSSIMV20(canvas, target)
			
			if newSSIM > initialSSIM + improvementThreshold {
				// Keep it removed
				lines = append(lines[:i], lines[i+1:]...)
				removed++
				passRemoved++
				initialSSIM = newSSIM
				fmt.Printf("Removed line %d->%d (SSIM: %.4f)\n", line.From, line.To, newSSIM)
			} else {
				// Put it back
				applyLineToCanvasV20(canvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight))
			}
		}
		
		if passRemoved == 0 {
			break // No more improvements found
		}
	}

	fmt.Printf("Phase 2 complete: %d lines removed\n", removed)

	// Phase 3: Strategic line addition for SSIM improvement
	fmt.Println("\n--- Phase 3: Strategic Line Addition ---")
	added := 0
	maxAdditions := 100

	for added < maxAdditions {
		bestAddition := Line{From: -1, To: -1, Score: -1}
		
		// Try adding lines between high-importance areas
		for fromPin := 0; fromPin < len(pins) && added < maxAdditions; fromPin++ {
			for toPin := 0; toPin < len(pins); toPin++ {
				if fromPin == toPin {
					continue
				}

				key := [2]int{fromPin, toPin}
				pixels, exists := linePixels[key]
				if !exists {
					continue
				}

				// Check if this line would improve SSIM
				applyLineToCanvasV20(canvas, pixels, float64(config.LineWeight))
				newSSIM := computeCanvasSSIMV20(canvas, target)
				removeLineFromCanvasV20(canvas, pixels, float64(config.LineWeight))

				improvement := newSSIM - initialSSIM
				if improvement > bestAddition.Score {
					bestAddition = Line{From: fromPin, To: toPin, Score: improvement}
				}
			}
		}

		// If we found a beneficial line, add it
		if bestAddition.Score > 0.0003 {
			applyLineToCanvasV20(canvas, linePixels[[2]int{bestAddition.From, bestAddition.To}], float64(config.LineWeight))
			lines = append(lines, bestAddition)
			added++
			initialSSIM += bestAddition.Score
			fmt.Printf("Added strategic line %d->%d (SSIM: %.4f)\n", bestAddition.From, bestAddition.To, initialSSIM)
		} else {
			break
		}
	}

	fmt.Printf("Phase 3 complete: %d lines added\n", added)

	finalSSIM := computeCanvasSSIMV20(canvas, target)
	finalMSE := computeWeightedMSEV20(canvas, target, importance)
	fmt.Printf("Final SSIM: %.4f\n", finalSSIM)
	fmt.Printf("Final MSE: %.1f\n", finalMSE)
	fmt.Printf("Total lines: %d\n", len(lines))

	return lines, canvas
}

// createV20ImportanceMap creates an enhanced importance map with better face detection
func createV20ImportanceMap(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	// Base importance from edge map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + 3.0*edgeStrength // Higher edge weight than v5
		}
	}

	// Enhanced face detection for cat features
	centerX, centerY := width/2, height/2
	faceRadius := int(float64(math.Min(float64(width), float64(height))) * 0.35)

	// Eye regions (cats have eyes in upper area, slightly apart)
	eyeY := centerY - faceRadius/3
	leftEyeX := centerX - faceRadius/4
	rightEyeX := centerX + faceRadius/4
	eyeRadius := faceRadius / 6

	// Nose region (center, below eyes)
	noseY := centerY + faceRadius/8
	noseRadius := faceRadius / 12

	// Ear regions (upper corners)
	earY := centerY - faceRadius/2
	leftEarX := centerX - faceRadius/2
	rightEarX := centerX + faceRadius/2
	earRadius := faceRadius / 8

	// Boost importance around facial features
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Left eye region
			leftEyeDist := math.Sqrt(float64((x-leftEyeX)*(x-leftEyeX) + (y-eyeY)*(y-eyeY)))
			if leftEyeDist < float64(eyeRadius) {
				importance[y][x] *= 2.5
			}

			// Right eye region
			rightEyeDist := math.Sqrt(float64((x-rightEyeX)*(x-rightEyeX) + (y-eyeY)*(y-eyeY)))
			if rightEyeDist < float64(eyeRadius) {
				importance[y][x] *= 2.5
			}

			// Nose region
			noseDist := math.Sqrt(float64((x-centerX)*(x-centerX) + (y-noseY)*(y-noseY)))
			if noseDist < float64(noseRadius) {
				importance[y][x] *= 2.0
			}

			// Left ear region
			leftEarDist := math.Sqrt(float64((x-leftEarX)*(x-leftEarX) + (y-earY)*(y-earY)))
			if leftEarDist < float64(earRadius) {
				importance[y][x] *= 1.8
			}

			// Right ear region
			rightEarDist := math.Sqrt(float64((x-rightEarX)*(x-rightEarX) + (y-earY)*(y-earY)))
			if rightEarDist < float64(earRadius) {
				importance[y][x] *= 1.8
			}
		}
	}

	return importance
}

// findBestLineV20 finds the best line using enhanced scoring (based on v5 but improved)
func findBestLineV20(currentPin int, pins []Pin, canvas [][]float64, target [][]float64,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, 
	weight float64, usedLines map[[2]int]int) Line {

	bestScore := -1.0
	bestTo := -1

	// Evaluate all possible next pins
	for nextPin := 0; nextPin < len(pins); nextPin++ {
		if nextPin == currentPin {
			continue
		}

		key := [2]int{currentPin, nextPin}
		pixels, exists := linePixels[key]
		if !exists {
			continue
		}

		// Penalize overused lines more strictly
		usageCount := usedLines[key]
		if usageCount >= 2 {
			continue
		}

		// Compute enhanced score for this line (similar to v5 but with importance weighting)
		score := computeLineScoreV20(canvas, target, importance, pixels, weight)
		
		// Apply usage penalty
		if usageCount > 0 {
			score *= 0.8
		}

		if score > bestScore {
			bestScore = score
			bestTo = nextPin
		}
	}

	return Line{From: currentPin, To: bestTo, Score: bestScore}
}

// computeLineScoreV20 computes enhanced score for a potential line (improved from v5)
func computeLineScoreV20(canvas [][]float64, target [][]float64, importance [][]float64,
	pixels []AntiAliasedPixel, weight float64) float64 {

	if len(pixels) == 0 {
		return 0
	}

	totalScore := 0.0
	
	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			currentValue := canvas[pixel.Y][pixel.X]
			targetValue := target[pixel.Y][pixel.X]
			
			// Compute how much this pixel would be darkened
			darkness := weight * pixel.Weight * 0.15 // Calibrated alpha
			newValue := currentValue - darkness
			if newValue < 0 {
				newValue = 0
			}
			
			// Score based on how much closer we get to target
			currentError := math.Abs(currentValue - targetValue)
			newError := math.Abs(newValue - targetValue)
			improvement := currentError - newError
			
			// Weight by importance and pixel weight
			imp := importance[pixel.Y][pixel.X]
			totalScore += improvement * imp * pixel.Weight
		}
	}

	return totalScore
}

// applyLineToCanvasV20 applies a line to the canvas (same as v5 but with calibrated alpha)
func applyLineToCanvasV20(canvas [][]float64, pixels []AntiAliasedPixel, weight float64) {
	alpha := 0.15 // Calibrated for mobile SVG rendering
	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			darkness := weight * pixel.Weight * alpha
			canvas[pixel.Y][pixel.X] -= darkness
			if canvas[pixel.Y][pixel.X] < 0 {
				canvas[pixel.Y][pixel.X] = 0
			}
		}
	}
}

// removeLineFromCanvasV20 removes a line from the canvas
func removeLineFromCanvasV20(canvas [][]float64, pixels []AntiAliasedPixel, weight float64) {
	alpha := 0.15 // Calibrated for mobile SVG rendering
	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			darkness := weight * pixel.Weight * alpha
			canvas[pixel.Y][pixel.X] += darkness
			if canvas[pixel.Y][pixel.X] > 255 {
				canvas[pixel.Y][pixel.X] = 255
			}
		}
	}
}

// computeCanvasSSIMV20 computes SSIM between canvas and target
func computeCanvasSSIMV20(canvas [][]float64, target [][]float64) float64 {
	height := len(target)
	width := len(target[0])
	
	// Convert to 1D arrays for SSIM computation
	img1 := make([]float64, width*height)
	img2 := make([]float64, width*height)
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img1[y*width+x] = canvas[y][x]
			img2[y*width+x] = target[y][x]
		}
	}
	
	return computeSSIMV20(img1, img2, width, height)
}

// computeWeightedMSEV20 computes weighted MSE between canvas and target
func computeWeightedMSEV20(canvas [][]float64, target [][]float64, importance [][]float64) float64 {
	height := len(target)
	width := len(target[0])
	
	totalError := 0.0
	totalWeight := 0.0
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			error := (canvas[y][x] - target[y][x]) * (canvas[y][x] - target[y][x])
			weight := importance[y][x]
			totalError += error * weight
			totalWeight += weight
		}
	}
	
	if totalWeight == 0 {
		return 0
	}
	
	return totalError / totalWeight
}

// computeSSIMV20 computes Structural Similarity Index between two images
func computeSSIMV20(img1, img2 []float64, width, height int) float64 {
	if len(img1) != len(img2) || len(img1) != width*height {
		return 0.0
	}

	// SSIM constants
	c1 := 6.5025   // (0.01 * 255)^2
	c2 := 58.5225  // (0.03 * 255)^2

	// Compute means
	mu1 := 0.0
	mu2 := 0.0
	for i := 0; i < len(img1); i++ {
		mu1 += img1[i]
		mu2 += img2[i]
	}
	mu1 /= float64(len(img1))
	mu2 /= float64(len(img2))

	// Compute variances and covariance
	sigma1Sq := 0.0
	sigma2Sq := 0.0
	sigma12 := 0.0

	for i := 0; i < len(img1); i++ {
		diff1 := img1[i] - mu1
		diff2 := img2[i] - mu2
		sigma1Sq += diff1 * diff1
		sigma2Sq += diff2 * diff2
		sigma12 += diff1 * diff2
	}

	n := float64(len(img1))
	sigma1Sq /= n
	sigma2Sq /= n
	sigma12 /= n

	// Compute SSIM
	numerator := (2*mu1*mu2 + c1) * (2*sigma12 + c2)
	denominator := (mu1*mu1 + mu2*mu2 + c1) * (sigma1Sq + sigma2Sq + c2)

	if denominator == 0 {
		return 1.0
	}

	return numerator / denominator
}