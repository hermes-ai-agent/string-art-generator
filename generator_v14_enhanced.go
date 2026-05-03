package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV14Enhanced implements focused improvements over V13:
// 1. Enhanced face feature detection with better importance mapping
// 2. Improved add/remove optimization with multiple passes
// 3. Better source-over alpha calibration for mobile SVG rendering
// 4. Perceptual scoring improvements
func GenerateStringArtV14Enhanced(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v14.0 Enhanced ===\n")
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
	importance := createSophisticatedImportanceMap(img, edgeMap, width, height)

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

		bestLine := findBestLineV14Enhanced(currentPin, pins, canvas, target, edgeMap, importance,
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
				alpha := (adaptiveWeight / 255.0) * 0.92 // Better calibration factor
				canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[key]++

		if (i+1)%200 == 0 {
			mse := calculateMSEV14Enhanced(canvas, target, width, height)
			ssim := calculateSSIMV14Enhanced(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM: %.4f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, adaptiveWeight)
		}
	}

	// Phase 2: Enhanced Multi-Pass Line Removal with SSIM optimization
	fmt.Println("\n--- Phase 2: Enhanced Multi-Pass Line Removal ---")
	beforeSSIM := calculateSSIMV14Enhanced(canvas, target, width, height)
	fmt.Printf("Before removal: SSIM = %.4f\n", beforeSSIM)

	removedCount := 0
	maxRemovalPasses := 3
	
	for pass := 0; pass < maxRemovalPasses; pass++ {
		fmt.Printf("Removal pass %d/%d...\n", pass+1, maxRemovalPasses)
		passRemovals := 0
		
		// Try removing lines in different orders for better optimization
		indices := make([]int, len(lines))
		for i := range indices {
			indices[i] = i
		}
		
		// Alternate between forward and backward passes
		if pass%2 == 1 {
			for i := 0; i < len(indices)/2; i++ {
				indices[i], indices[len(indices)-1-i] = indices[len(indices)-1-i], indices[i]
			}
		}
		
		for _, i := range indices {
			if i >= len(lines) {
				continue
			}
			
			line := lines[i]
			key := [2]int{min(line.From, line.To), max(line.From, line.To)}
			pixels := linePixels[key]

			// Temporarily remove line
			for _, pixel := range pixels {
				if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
					alpha := (baseWeight / 255.0) * 0.92
					if 1.0-alpha*pixel.Weight > 0.001 {
						canvas[pixel.Y][pixel.X] = (canvas[pixel.Y][pixel.X] - 0.0*alpha*pixel.Weight) / (1.0-alpha*pixel.Weight)
					}
				}
			}

			// Check if removal improves SSIM
			testSSIM := calculateSSIMV14Enhanced(canvas, target, width, height)
			if testSSIM > beforeSSIM + 0.0008 { // Slightly higher threshold for better quality
				// Keep it removed
				lines = append(lines[:i], lines[i+1:]...)
				removedCount++
				passRemovals++
				beforeSSIM = testSSIM
				
				// Adjust indices for removed element
				for j := range indices {
					if indices[j] > i {
						indices[j]--
					}
				}
			} else {
				// Re-add the line
				for _, pixel := range pixels {
					if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
						alpha := (baseWeight / 255.0) * 0.92
						canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
					}
				}
			}
		}
		
		fmt.Printf("Pass %d: removed %d lines\n", pass+1, passRemovals)
		if passRemovals == 0 {
			break // No more improvements possible
		}
	}

	afterSSIM := calculateSSIMV14Enhanced(canvas, target, width, height)
	afterMSE := calculateMSEV14Enhanced(canvas, target, width, height)
	fmt.Printf("After removal: SSIM = %.4f, MSE = %.1f (removed %d lines)\n", afterSSIM, afterMSE, removedCount)

	fmt.Printf("Final: %d lines, SSIM = %.4f\n", len(lines), afterSSIM)
	return lines, canvas
}

// createSophisticatedImportanceMap creates an enhanced importance map with sophisticated face detection
func createSophisticatedImportanceMap(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	// Base importance from edge map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)
			importance[y][x] = 1.0 + edgeVal/255.0*2.5 // Slightly higher edge weight
		}
	}

	// Sophisticated face feature detection
	centerX, centerY := width/2, height/2
	
	// Eye regions (upper third, left and right) - more precise positioning
	eyeY := centerY - height/5 // Slightly higher
	leftEyeX := centerX - width/7 // Slightly closer
	rightEyeX := centerX + width/7
	eyeRadius := width / 18 // Slightly smaller, more focused
	
	// Nose region (center) - more precise
	noseY := centerY + height/20 // Slightly lower
	noseRadius := width / 28
	
	// Mouth region (lower third) - more precise
	mouthY := centerY + height/6 // Slightly higher
	mouthRadius := width / 20
	
	// Ear regions (sides)
	leftEarX := centerX - width/3
	rightEarX := centerX + width/3
	earY := centerY - height/10
	earRadius := width / 25
	
	// Apply enhanced weights to face features with gradual falloff
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Left eye with gradual falloff
			eyeDist := distance(x, y, leftEyeX, eyeY)
			if eyeDist < eyeRadius {
				falloff := 1.0 - float64(eyeDist)/float64(eyeRadius)
				importance[y][x] *= (1.0 + 3.0*falloff)
			}
			
			// Right eye with gradual falloff
			eyeDist = distance(x, y, rightEyeX, eyeY)
			if eyeDist < eyeRadius {
				falloff := 1.0 - float64(eyeDist)/float64(eyeRadius)
				importance[y][x] *= (1.0 + 3.0*falloff)
			}
			
			// Nose with gradual falloff
			noseDist := distance(x, y, centerX, noseY)
			if noseDist < noseRadius {
				falloff := 1.0 - float64(noseDist)/float64(noseRadius)
				importance[y][x] *= (1.0 + 2.2*falloff)
			}
			
			// Mouth with gradual falloff
			mouthDist := distance(x, y, centerX, mouthY)
			if mouthDist < mouthRadius {
				falloff := 1.0 - float64(mouthDist)/float64(mouthRadius)
				importance[y][x] *= (1.0 + 2.0*falloff)
			}
			
			// Left ear
			earDist := distance(x, y, leftEarX, earY)
			if earDist < earRadius {
				falloff := 1.0 - float64(earDist)/float64(earRadius)
				importance[y][x] *= (1.0 + 1.5*falloff)
			}
			
			// Right ear
			earDist = distance(x, y, rightEarX, earY)
			if earDist < earRadius {
				falloff := 1.0 - float64(earDist)/float64(earRadius)
				importance[y][x] *= (1.0 + 1.5*falloff)
			}
		}
	}

	return importance
}

// findBestLineV14Enhanced finds the best line using enhanced scoring
func findBestLineV14Enhanced(currentPin int, pins []Pin, canvas [][]float64, target [][]float64, edgeMap *image.Gray, importance [][]float64,
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

				score := calculateEnhancedLineScore(pixels, canvas, target, importance, lineWeight)
				
				// Apply usage penalty
				if usedLines[key] > 0 {
					score *= 0.75 // Slightly less penalty for better exploration
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

// calculateEnhancedLineScore calculates enhanced score for a potential line
func calculateEnhancedLineScore(pixels []AntiAliasedPixel, canvas, target, importance [][]float64, weight float64) float64 {
	if len(pixels) == 0 {
		return 0
	}

	score := 0.0
	count := 0
	alpha := (weight / 255.0) * 0.92

	for _, pixel := range pixels {
		if pixel.X >= 0 && pixel.X < len(canvas[0]) && pixel.Y >= 0 && pixel.Y < len(canvas) {
			currentVal := canvas[pixel.Y][pixel.X]
			targetVal := target[pixel.Y][pixel.X]
			
			// Calculate what the new value would be
			newVal := currentVal*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			
			// Enhanced scoring: combination of error reduction and perceptual factors
			currentError := math.Abs(currentVal - targetVal)
			newError := math.Abs(newVal - targetVal)
			improvement := currentError - newError
			
			// Perceptual enhancement: favor darker areas more (better contrast)
			perceptualWeight := 1.0
			if targetVal < 128 { // Dark areas
				perceptualWeight = 1.3
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

// calculateMSEV14Enhanced calculates MSE between canvas and target
func calculateMSEV14Enhanced(canvas, target [][]float64, width, height int) float64 {
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

// calculateSSIMV14Enhanced calculates SSIM between canvas and target
func calculateSSIMV14Enhanced(canvas, target [][]float64, width, height int) float64 {
	windowSize := 11
	halfWindow := windowSize / 2
	
	var ssimSum float64
	count := 0
	
	for y := halfWindow; y < height-halfWindow; y++ {
		for x := halfWindow; x < width-halfWindow; x++ {
			localSSIM := calculateLocalSSIMV14(canvas, target, x, y, halfWindow)
			ssimSum += localSSIM
			count++
		}
	}
	
	if count > 0 {
		return ssimSum / float64(count)
	}
	return 0
}

// calculateLocalSSIMV14 calculates SSIM in a local window
func calculateLocalSSIMV14(img1, img2 [][]float64, centerX, centerY, radius int) float64 {
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