package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV14Fast implements focused improvements over V13 with faster optimization:
// 1. Enhanced face feature detection with better importance mapping
// 2. Single-pass line removal optimization
// 3. Better source-over alpha calibration for mobile SVG rendering
// 4. Perceptual scoring improvements
func GenerateStringArtV14Fast(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v14.0 Fast ===\n")
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

	// Create enhanced importance map with sophisticated face detection
	importance := createOptimizedImportanceMap(img, edgeMap, width, height)

	// Pre-compute all line pixels with anti-aliasing
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)
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
		// Improved adaptive line weight with better calibration for mobile rendering
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.45*progress) // Slightly more aggressive reduction
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV14Fast(currentPin, pins, canvas, target, edgeMap, importance,
			linePixels, config, adaptiveWeight, usedLines)

		if bestLine.Score <= 0.05 { // Slightly higher threshold
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
			if avgScore < config.StopThreshold*0.25 { // More sensitive
				stagnationCount++
				if stagnationCount > 15 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Apply line to canvas with enhanced calibration for mobile rendering match
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				// Enhanced source-over compositing calibrated for mobile SVG rendering
				alpha := (adaptiveWeight / 255.0) * 0.94 // Optimized calibration factor
				canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[key]++

		if (i+1)%200 == 0 {
			mse := calculateMSEV14Fast(canvas, target, width, height)
			ssim := calculateSSIMV14Fast(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM: %.4f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, adaptiveWeight)
		}
	}

	// Phase 2: Fast Single-Pass Line Removal with SSIM optimization
	fmt.Println("\n--- Phase 2: Fast Line Removal ---")
	beforeSSIM := calculateSSIMV14Fast(canvas, target, width, height)
	fmt.Printf("Before removal: SSIM = %.4f\n", beforeSSIM)

	removedCount := 0
	
	// Single pass removal - check every 10th line for efficiency
	for i := len(lines) - 1; i >= 0; i -= 10 {
		if i >= len(lines) {
			continue
		}
		
		line := lines[i]
		key := [2]int{min(line.From, line.To), max(line.From, line.To)}
		pixels := linePixels[key]

		// Temporarily remove line
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				alpha := (baseWeight / 255.0) * 0.94
				if 1.0-alpha*pixel.Weight > 0.001 {
					canvas[pixel.Y][pixel.X] = (canvas[pixel.Y][pixel.X] - 0.0*alpha*pixel.Weight) / (1.0-alpha*pixel.Weight)
				}
			}
		}

		// Check if removal improves SSIM
		testSSIM := calculateSSIMV14Fast(canvas, target, width, height)
		if testSSIM > beforeSSIM + 0.001 { // Higher threshold for better quality
			// Keep it removed
			lines = append(lines[:i], lines[i+1:]...)
			removedCount++
			beforeSSIM = testSSIM
		} else {
			// Re-add the line
			for _, pixel := range pixels {
				if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
					alpha := (baseWeight / 255.0) * 0.94
					canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
				}
			}
		}
	}

	afterSSIM := calculateSSIMV14Fast(canvas, target, width, height)
	afterMSE := calculateMSEV14Fast(canvas, target, width, height)
	fmt.Printf("After removal: SSIM = %.4f, MSE = %.1f (removed %d lines)\n", afterSSIM, afterMSE, removedCount)

	fmt.Printf("Final: %d lines, SSIM = %.4f\n", len(lines), afterSSIM)
	return lines, canvas
}

// createOptimizedImportanceMap creates an optimized importance map with face detection
func createOptimizedImportanceMap(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	// Base importance from edge map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)
			importance[y][x] = 1.0 + edgeVal/255.0*2.8 // Higher edge weight
		}
	}

	// Optimized face feature detection
	centerX, centerY := width/2, height/2
	
	// Eye regions (upper third, left and right)
	eyeY := centerY - height/5
	leftEyeX := centerX - width/7
	rightEyeX := centerX + width/7
	eyeRadius := width / 16 // Larger for better coverage
	
	// Nose region (center)
	noseY := centerY + height/20
	noseRadius := width / 25
	
	// Mouth region (lower third)
	mouthY := centerY + height/6
	mouthRadius := width / 18
	
	// Apply enhanced weights to face features
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Left eye
			if distance(x, y, leftEyeX, eyeY) < eyeRadius {
				importance[y][x] *= 4.0 // Higher weight
			}
			// Right eye
			if distance(x, y, rightEyeX, eyeY) < eyeRadius {
				importance[y][x] *= 4.0 // Higher weight
			}
			// Nose
			if distance(x, y, centerX, noseY) < noseRadius {
				importance[y][x] *= 3.0
			}
			// Mouth
			if distance(x, y, centerX, mouthY) < mouthRadius {
				importance[y][x] *= 2.8
			}
		}
	}

	return importance
}

// findBestLineV14Fast finds the best line using optimized scoring
func findBestLineV14Fast(currentPin int, pins []Pin, canvas [][]float64, target [][]float64, edgeMap *image.Gray, importance [][]float64,
	linePixels map[[2]int][]AntiAliasedPixel, config *Config, lineWeight float64, usedLines map[[2]int]int) Line {

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

				score := calculateOptimizedLineScore(pixels, canvas, target, importance, lineWeight)
				
				// Apply usage penalty
				if usedLines[key] > 0 {
					score *= 0.8 // Moderate penalty
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

// calculateOptimizedLineScore calculates optimized score for a potential line
func calculateOptimizedLineScore(pixels []AntiAliasedPixel, canvas, target, importance [][]float64, weight float64) float64 {
	if len(pixels) == 0 {
		return 0
	}

	score := 0.0
	count := 0
	alpha := (weight / 255.0) * 0.94

	for _, pixel := range pixels {
		if pixel.X >= 0 && pixel.X < len(canvas[0]) && pixel.Y >= 0 && pixel.Y < len(canvas) {
			currentVal := canvas[pixel.Y][pixel.X]
			targetVal := target[pixel.Y][pixel.X]
			
			// Calculate what the new value would be
			newVal := currentVal*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			
			// Optimized scoring: focus on error reduction with perceptual enhancement
			currentError := math.Abs(currentVal - targetVal)
			newError := math.Abs(newVal - targetVal)
			improvement := currentError - newError
			
			// Enhanced perceptual weighting
			perceptualWeight := 1.0
			if targetVal < 100 { // Very dark areas
				perceptualWeight = 1.5
			} else if targetVal < 150 { // Medium dark areas
				perceptualWeight = 1.2
			}
			
			// Weight by importance, pixel weight, and perceptual factors
			importanceWeight := importance[pixel.Y][pixel.X]
			score += improvement * importanceWeight * pixel.Weight * perceptualWeight
			count++
		}
	}

	if count > 0 {
		return score / float64(count)
	}
	return 0
}

// calculateMSEV14Fast calculates MSE between canvas and target
func calculateMSEV14Fast(canvas, target [][]float64, width, height int) float64 {
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

// calculateSSIMV14Fast calculates SSIM between canvas and target
func calculateSSIMV14Fast(canvas, target [][]float64, width, height int) float64 {
	windowSize := 11
	halfWindow := windowSize / 2
	
	var ssimSum float64
	count := 0
	
	for y := halfWindow; y < height-halfWindow; y++ {
		for x := halfWindow; x < width-halfWindow; x++ {
			localSSIM := calculateLocalSSIMV14Fast(canvas, target, x, y, halfWindow)
			ssimSum += localSSIM
			count++
		}
	}
	
	if count > 0 {
		return ssimSum / float64(count)
	}
	return 0
}

// calculateLocalSSIMV14Fast calculates SSIM in a local window
func calculateLocalSSIMV14Fast(img1, img2 [][]float64, centerX, centerY, radius int) float64 {
	var sum1, sum2, sum1Sq, sum2Sq, sum12 float64
	count := 0
	
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			x, y := centerX+dx, centerY+dy
			if y >= 0 && y < len(img1) && x >= 0 && x < len(img1[0]) {
				v1 := img1[y][x]
				v2 := img2[y][x]
				sum1 += v1
				sum2 += v2
				sum1Sq += v1 * v1
				sum2Sq += v2 * v2
				sum12 += v1 * v2
				count++
			}
		}
	}
	
	if count == 0 {
		return 0
	}
	
	mu1 := sum1 / float64(count)
	mu2 := sum2 / float64(count)
	mu1Sq := mu1 * mu1
	mu2Sq := mu2 * mu2
	mu1Mu2 := mu1 * mu2
	
	sigma1Sq := sum1Sq/float64(count) - mu1Sq
	sigma2Sq := sum2Sq/float64(count) - mu2Sq
	sigma12 := sum12/float64(count) - mu1Mu2
	
	C1 := (0.01 * 255) * (0.01 * 255)
	C2 := (0.03 * 255) * (0.03 * 255)
	
	numerator := (2*mu1Mu2 + C1) * (2*sigma12 + C2)
	denominator := (mu1Sq + mu2Sq + C1) * (sigma1Sq + sigma2Sq + C2)
	
	if denominator == 0 {
		return 0
	}
	
	return numerator / denominator
}