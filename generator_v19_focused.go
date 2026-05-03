package main

import (
	"fmt"
	"image"
	"math"
)

// GenerateStringArtV19Focused implements focused improvements for v3.3.0+ requirements:
// 1. Enhanced importance mapping with better face detection
// 2. Improved add/remove optimization with SSIM-based scoring
// 3. Better calibration for mobile SVG rendering
// 4. Perceptual scoring improvements
func GenerateStringArtV19Focused(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v19.0 Focused ===\n")
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
	importance := createV19ImportanceMap(img, edgeMap, width, height)

	// Pre-compute all line pixels with anti-aliasing
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Adaptive parameters
	baseWeight := float64(config.LineWeight)

	// Phase 1: Greedy line addition with enhanced scoring
	fmt.Println("\n--- Phase 1: Greedy Line Addition ---")
	for i := 0; i < config.NumLines; i++ {
		// Adaptive line weight - start higher, reduce more gradually
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.3*progress)
		if adaptiveWeight < 15 {
			adaptiveWeight = 15
		}

		bestLine := findBestLineV19(currentPin, pins, canvas, target, importance,
			linePixels, adaptiveWeight, usedLines)

		if bestLine.Score <= 0.1 {
			fmt.Printf("Stopping at line %d (no improvement possible, score: %.4f)\n", i, bestLine.Score)
			break
		}

		// Apply the line to canvas
		applyLineToCanvasV19(canvas, linePixels[[2]int{bestLine.From, bestLine.To}], adaptiveWeight)

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[[2]int{bestLine.From, bestLine.To}]++

		if (i+1)%200 == 0 {
			// Compute current metrics for progress tracking
			currentSSIM := computeCanvasSSIM(canvas, target)
			currentMSE := computeWeightedMSE(canvas, target, importance)
			fmt.Printf("Progress: %d/%d lines (SSIM: %.4f, MSE: %.1f, weight: %.1f)\n", 
				i+1, config.NumLines, currentSSIM, currentMSE, adaptiveWeight)
		}
	}

	fmt.Printf("Phase 1 complete: %d lines added\n", len(lines))

	// Phase 2: Enhanced Add/Remove optimization with SSIM-based scoring
	fmt.Println("\n--- Phase 2: Enhanced Add/Remove Optimization ---")
	initialSSIM := computeCanvasSSIM(canvas, target)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	// Try removing lines that hurt quality (SSIM-based)
	removed := 0
	improved := true
	maxRemovals := 100

	for improved && removed < maxRemovals {
		improved = false
		bestRemovalIndex := -1
		bestSSIMAfterRemoval := initialSSIM

		// Test removing each line (focus on recent lines first)
		startIndex := len(lines) - 1
		endIndex := 0
		if len(lines) > 100 {
			endIndex = len(lines) - 100 // Only test last 100 lines
		}

		for i := startIndex; i >= endIndex; i-- {
			line := lines[i]
			
			// Remove line temporarily
			removeLineFromCanvasV19(canvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight))
			
			// Check if SSIM improved
			newSSIM := computeCanvasSSIM(canvas, target)
			
			if newSSIM > bestSSIMAfterRemoval {
				bestSSIMAfterRemoval = newSSIM
				bestRemovalIndex = i
			}
			
			// Put it back for now
			applyLineToCanvasV19(canvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight))
		}

		// If we found an improvement, remove the best line
		if bestRemovalIndex >= 0 && bestSSIMAfterRemoval > initialSSIM + 0.0005 {
			line := lines[bestRemovalIndex]
			removeLineFromCanvasV19(canvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight))
			lines = append(lines[:bestRemovalIndex], lines[bestRemovalIndex+1:]...)
			removed++
			initialSSIM = bestSSIMAfterRemoval
			improved = true
			fmt.Printf("Removed line %d->%d (SSIM: %.4f)\n", line.From, line.To, bestSSIMAfterRemoval)
		}
	}

	fmt.Printf("Phase 2 complete: %d lines removed\n", removed)

	// Phase 3: Try adding new beneficial lines
	fmt.Println("\n--- Phase 3: Beneficial Line Addition ---")
	added := 0
	maxAdditions := 50

	for added < maxAdditions {
		bestAddition := Line{From: -1, To: -1, Score: -1}
		
		// Try adding lines from all pins
		for fromPin := 0; fromPin < len(pins); fromPin++ {
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
				applyLineToCanvasV19(canvas, pixels, float64(config.LineWeight))
				newSSIM := computeCanvasSSIM(canvas, target)
				removeLineFromCanvasV19(canvas, pixels, float64(config.LineWeight))

				improvement := newSSIM - initialSSIM
				if improvement > bestAddition.Score {
					bestAddition = Line{From: fromPin, To: toPin, Score: improvement}
				}
			}
		}

		// If we found a beneficial line, add it
		if bestAddition.Score > 0.0005 {
			applyLineToCanvasV19(canvas, linePixels[[2]int{bestAddition.From, bestAddition.To}], float64(config.LineWeight))
			lines = append(lines, bestAddition)
			added++
			initialSSIM += bestAddition.Score
			fmt.Printf("Added beneficial line %d->%d (SSIM: %.4f)\n", bestAddition.From, bestAddition.To, initialSSIM)
		} else {
			break
		}
	}

	fmt.Printf("Phase 3 complete: %d lines added\n", added)

	finalSSIM := computeCanvasSSIM(canvas, target)
	finalMSE := computeWeightedMSE(canvas, target, importance)
	fmt.Printf("Final SSIM: %.4f\n", finalSSIM)
	fmt.Printf("Final MSE: %.1f\n", finalMSE)
	fmt.Printf("Total lines: %d\n", len(lines))

	return lines, canvas
}

// createV19ImportanceMap creates an enhanced importance map with better face detection
func createV19ImportanceMap(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	// Base importance from edge map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + 2.5*edgeStrength // Higher edge weight
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
				importance[y][x] *= 3.0
			}

			// Right eye region
			rightEyeDist := math.Sqrt(float64((x-rightEyeX)*(x-rightEyeX) + (y-eyeY)*(y-eyeY)))
			if rightEyeDist < float64(eyeRadius) {
				importance[y][x] *= 3.0
			}

			// Nose region
			noseDist := math.Sqrt(float64((x-centerX)*(x-centerX) + (y-noseY)*(y-noseY)))
			if noseDist < float64(noseRadius) {
				importance[y][x] *= 2.5
			}

			// Left ear region
			leftEarDist := math.Sqrt(float64((x-leftEarX)*(x-leftEarX) + (y-earY)*(y-earY)))
			if leftEarDist < float64(earRadius) {
				importance[y][x] *= 2.0
			}

			// Right ear region
			rightEarDist := math.Sqrt(float64((x-rightEarX)*(x-rightEarX) + (y-earY)*(y-earY)))
			if rightEarDist < float64(earRadius) {
				importance[y][x] *= 2.0
			}
		}
	}

	return importance
}

// findBestLineV19 finds the best line using enhanced perceptual scoring
func findBestLineV19(currentPin int, pins []Pin, canvas [][]float64, target [][]float64,
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

		// Penalize overused lines
		usageCount := usedLines[key]
		if usageCount >= 2 {
			continue
		}

		// Compute enhanced perceptual score for this line
		score := computeLineEnhancedScore(canvas, target, importance, pixels, weight)
		
		// Apply usage penalty
		if usageCount > 0 {
			score *= 0.85
		}

		if score > bestScore {
			bestScore = score
			bestTo = nextPin
		}
	}

	return Line{From: currentPin, To: bestTo, Score: bestScore}
}

// computeLineEnhancedScore computes enhanced perceptual score for a potential line
func computeLineEnhancedScore(canvas [][]float64, target [][]float64, importance [][]float64,
	pixels []AntiAliasedPixel, weight float64) float64 {

	if len(pixels) == 0 {
		return 0
	}

	// Compute weighted error improvement for affected pixels
	totalImprovement := 0.0
	totalWeight := 0.0

	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			// Current error
			currentValue := canvas[pixel.Y][pixel.X]
			targetValue := target[pixel.Y][pixel.X]
			currentError := math.Abs(currentValue - targetValue)
			
			// New value after applying line
			darkness := weight * pixel.Weight * 0.15 // Calibrated alpha
			newValue := currentValue - darkness
			if newValue < 0 {
				newValue = 0
			}
			newError := math.Abs(newValue - targetValue)
			
			// Improvement (positive means better)
			improvement := currentError - newError
			
			// Weight by importance and pixel weight
			imp := importance[pixel.Y][pixel.X]
			pixelWeight := pixel.Weight
			totalImprovement += improvement * imp * pixelWeight
			totalWeight += imp * pixelWeight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return totalImprovement / totalWeight
}

// applyLineToCanvasV19 applies a line to the canvas
func applyLineToCanvasV19(canvas [][]float64, pixels []AntiAliasedPixel, weight float64) {
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

// removeLineFromCanvasV19 removes a line from the canvas
func removeLineFromCanvasV19(canvas [][]float64, pixels []AntiAliasedPixel, weight float64) {
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

// computeCanvasSSIM computes SSIM between canvas and target
func computeCanvasSSIM(canvas [][]float64, target [][]float64) float64 {
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
	
	return computeSSIMV19(img1, img2, width, height)
}

// computeWeightedMSE computes weighted MSE between canvas and target
func computeWeightedMSE(canvas [][]float64, target [][]float64, importance [][]float64) float64 {
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

// computeSSIMV19 computes Structural Similarity Index between two images
func computeSSIMV19(img1, img2 []float64, width, height int) float64 {
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