package main

import (
	"fmt"
	"image"
	"math"
)

// GenerateStringArtV17Enhanced implements focused improvements for v3.3.0+:
// 1. Birsak 2018 supersampled rendering (4x for better quality)
// 2. MSE-based scoring with importance weighting (more reliable than SSIM for single lines)
// 3. Add/Remove optimization after greedy phase
// 4. Enhanced importance map with face detection
// 5. Calibrated source-over alpha for mobile SVG matching
func GenerateStringArtV17Enhanced(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Use 4x supersampling for better quality (like Birsak 2018)
	supersample := 4
	superWidth := width * supersample
	superHeight := height * supersample

	fmt.Printf("=== String Art Generator v17.0 Enhanced ===\n")
	fmt.Printf("Base Resolution: %dx%d\n", width, height)
	fmt.Printf("Super Resolution: %dx%d (4x supersampling)\n", superWidth, superHeight)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)

	// Create target array at base resolution
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y)
		}
	}

	// Create supersampled canvas (starts white)
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
	importance := createV17ImportanceMap(img, edgeMap, width, height)

	// Pre-compute line pixels at supersampled resolution
	fmt.Println("Pre-computing supersampled line pixels...")
	linePixels := precomputeLinePixelsV17(pins, superWidth, superHeight, config.MinDistance*supersample)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Calibrated alpha for mobile SVG matching (based on baseline analysis)
	alpha := 0.08 // Lower alpha for 4x supersampling

	// Phase 1: Greedy line addition with MSE-based scoring
	fmt.Println("\n--- Phase 1: Greedy Line Addition (MSE-based) ---")
	for i := 0; i < config.NumLines; i++ {
		// Adaptive line weight
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := float64(config.LineWeight) * (1.0 - 0.2*progress)
		if adaptiveWeight < 18 {
			adaptiveWeight = 18
		}

		bestLine := findBestLineV17(currentPin, pins, superCanvas, target, importance,
			linePixels, adaptiveWeight, usedLines, supersample, alpha)

		if bestLine.Score <= 0.1 {
			fmt.Printf("Stopping at line %d (no improvement possible, score: %.4f)\n", i, bestLine.Score)
			break
		}

		// Apply the line to supersampled canvas
		applyLineToSuperCanvasV17(superCanvas, linePixels[[2]int{bestLine.From, bestLine.To}], adaptiveWeight, alpha)

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[[2]int{bestLine.From, bestLine.To}]++

		if (i+1)%200 == 0 {
			// Compute current MSE for progress tracking
			currentMSE := computeCurrentMSEV17(superCanvas, target, importance, supersample)
			fmt.Printf("Progress: %d/%d lines (MSE: %.2f, weight: %.1f)\n", 
				i+1, config.NumLines, currentMSE, adaptiveWeight)
		}
	}

	fmt.Printf("Phase 1 complete: %d lines added\n", len(lines))

	// Phase 2: Add/Remove optimization
	fmt.Println("\n--- Phase 2: Add/Remove Optimization ---")
	initialMSE := computeCurrentMSEV17(superCanvas, target, importance, supersample)
	fmt.Printf("Initial MSE: %.2f\n", initialMSE)

	// Try removing lines that hurt quality
	removed := 0
	for i := len(lines) - 1; i >= 0 && removed < 100; i-- {
		line := lines[i]
		
		// Remove line temporarily
		removeLineFromSuperCanvasV17(superCanvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight), alpha)
		
		// Check if quality improved
		newMSE := computeCurrentMSEV17(superCanvas, target, importance, supersample)
		
		if newMSE < initialMSE - 10.0 { // MSE improvement threshold
			// Keep it removed
			lines = append(lines[:i], lines[i+1:]...)
			removed++
			initialMSE = newMSE
			fmt.Printf("Removed line %d->%d (MSE: %.2f)\n", line.From, line.To, newMSE)
		} else {
			// Put it back
			applyLineToSuperCanvasV17(superCanvas, linePixels[[2]int{line.From, line.To}], float64(config.LineWeight), alpha)
		}
	}

	fmt.Printf("Phase 2 complete: %d lines removed\n", removed)

	// Downsample canvas to base resolution for final output
	canvas := downsampleCanvasV17(superCanvas, supersample)

	finalMSE := computeCurrentMSEV17(superCanvas, target, importance, supersample)
	finalSSIM := computeCurrentSSIMV17(superCanvas, target, supersample)
	fmt.Printf("Final MSE: %.2f\n", finalMSE)
	fmt.Printf("Final SSIM: %.4f\n", finalSSIM)
	fmt.Printf("Total lines: %d\n", len(lines))

	return lines, canvas
}

// createV17ImportanceMap creates an enhanced importance map with face detection
func createV17ImportanceMap(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	// Base importance from edge map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + 3.0*edgeStrength // Higher edge weight
		}
	}

	// Enhanced face detection - look for eye/nose regions
	centerX, centerY := width/2, height/2
	faceRadius := int(float64(math.Min(float64(width), float64(height))) * 0.35)

	// Eye regions (upper third of face area)
	eyeY := centerY - faceRadius/3
	leftEyeX := centerX - faceRadius/3
	rightEyeX := centerX + faceRadius/3
	eyeRadius := faceRadius / 5

	// Boost importance around likely eye positions
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

			// Nose region (center, slightly below eyes)
			noseY := centerY + faceRadius/6
			noseDist := math.Sqrt(float64((x-centerX)*(x-centerX) + (y-noseY)*(y-noseY)))
			if noseDist < float64(eyeRadius) {
				importance[y][x] *= 2.0
			}
		}
	}

	return importance
}

// precomputeLinePixelsV17 precomputes all valid line segments with anti-aliasing
func precomputeLinePixelsV17(pins []Pin, width, height, minDistance int) map[[2]int][]AntiAliasedPixel {
	numPins := len(pins)
	result := make(map[[2]int][]AntiAliasedPixel)

	for i := 0; i < numPins; i++ {
		for j := 0; j < numPins; j++ {
			if i == j {
				continue
			}

			// Check minimum distance constraint
			pinDistance := math.Abs(float64(i - j))
			if pinDistance > float64(numPins)/2 {
				pinDistance = float64(numPins) - pinDistance
			}
			if pinDistance < float64(minDistance) {
				continue
			}

			// Generate anti-aliased line pixels
			pixels := getAntiAliasedLinePixels(pins[i], pins[j], width, height)
			if len(pixels) > 0 {
				result[[2]int{i, j}] = pixels
			}
		}
	}

	return result
}

// findBestLineV17 finds the best line from current pin using MSE-based scoring
func findBestLineV17(currentPin int, pins []Pin, canvas [][]float64, target [][]float64,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel, 
	weight float64, usedLines map[[2]int]int, supersample int, alpha float64) Line {

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

		// Compute MSE-based score for this line
		score := computeLineMSEScore(canvas, target, importance, pixels, weight, supersample, alpha)
		
		// Apply usage penalty
		if usageCount > 0 {
			score *= 0.8 // Reduce score for reused lines
		}

		if score > bestScore {
			bestScore = score
			bestTo = nextPin
		}
	}

	return Line{From: currentPin, To: bestTo, Score: bestScore}
}

// computeLineMSEScore computes MSE-based score for a potential line
func computeLineMSEScore(canvas [][]float64, target [][]float64, importance [][]float64,
	pixels []AntiAliasedPixel, weight float64, supersample int, alpha float64) float64 {

	if len(pixels) == 0 {
		return 0
	}

	// Compute MSE improvement for affected pixels
	totalImprovement := 0.0
	affectedPixels := 0

	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			// Get corresponding target pixel (downsample coordinates)
			targetY := pixel.Y / supersample
			targetX := pixel.X / supersample
			
			if targetY >= 0 && targetY < len(target) && targetX >= 0 && targetX < len(target[0]) {
				// Current error
				currentValue := canvas[pixel.Y][pixel.X]
				targetValue := target[targetY][targetX]
				currentError := (currentValue - targetValue) * (currentValue - targetValue)
				
				// New value after applying line
				darkness := weight * pixel.Weight * alpha
				newValue := currentValue * (1.0 - alpha) + (currentValue - darkness) * alpha
				if newValue < 0 {
					newValue = 0
				}
				newError := (newValue - targetValue) * (newValue - targetValue)
				
				// Improvement (negative means better)
				improvement := currentError - newError
				
				// Weight by importance
				imp := importance[targetY][targetX]
				totalImprovement += improvement * imp
				affectedPixels++
			}
		}
	}

	if affectedPixels == 0 {
		return 0
	}

	return totalImprovement / float64(affectedPixels)
}

// applyLineToSuperCanvasV17 applies a line to the supersampled canvas
func applyLineToSuperCanvasV17(canvas [][]float64, pixels []AntiAliasedPixel, weight, alpha float64) {
	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			// Source-over blending: new = old * (1 - alpha) + new * alpha
			darkness := weight * pixel.Weight * alpha
			canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X] * (1.0 - alpha) + (canvas[pixel.Y][pixel.X] - darkness) * alpha
			if canvas[pixel.Y][pixel.X] < 0 {
				canvas[pixel.Y][pixel.X] = 0
			}
		}
	}
}

// removeLineFromSuperCanvasV17 removes a line from the supersampled canvas
func removeLineFromSuperCanvasV17(canvas [][]float64, pixels []AntiAliasedPixel, weight, alpha float64) {
	for _, pixel := range pixels {
		if pixel.Y >= 0 && pixel.Y < len(canvas) && pixel.X >= 0 && pixel.X < len(canvas[0]) {
			// Reverse the source-over blending (approximation)
			darkness := weight * pixel.Weight * alpha
			canvas[pixel.Y][pixel.X] += darkness * alpha
			if canvas[pixel.Y][pixel.X] > 255 {
				canvas[pixel.Y][pixel.X] = 255
			}
		}
	}
}

// computeCurrentMSEV17 computes MSE between current canvas and target
func computeCurrentMSEV17(superCanvas [][]float64, target [][]float64, importance [][]float64, supersample int) float64 {
	// Downsample canvas to match target resolution
	canvas := downsampleCanvasV17(superCanvas, supersample)
	
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

// computeCurrentSSIMV17 computes SSIM between current canvas and target
func computeCurrentSSIMV17(superCanvas [][]float64, target [][]float64, supersample int) float64 {
	// Downsample canvas to match target resolution
	canvas := downsampleCanvasV17(superCanvas, supersample)
	
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
	
	return computeSSIMV17(img1, img2, width, height)
}

// downsampleCanvasV17 downsamples supersampled canvas to base resolution
func downsampleCanvasV17(superCanvas [][]float64, supersample int) [][]float64 {
	superHeight := len(superCanvas)
	superWidth := len(superCanvas[0])
	height := superHeight / supersample
	width := superWidth / supersample

	canvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			// Average the supersample x supersample block
			sum := 0.0
			for sy := 0; sy < supersample; sy++ {
				for sx := 0; sx < supersample; sx++ {
					sum += superCanvas[y*supersample+sy][x*supersample+sx]
				}
			}
			canvas[y][x] = sum / float64(supersample*supersample)
		}
	}

	return canvas
}

// computeSSIMV17 computes Structural Similarity Index between two images
func computeSSIMV17(img1, img2 []float64, width, height int) float64 {
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