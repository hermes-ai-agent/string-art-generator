package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV13Optimized implements minimal but focused improvements over V5:
// 1. Better source-over alpha calibration to match mobile rendering
// 2. Enhanced importance mapping for face features
// 3. Improved line removal with SSIM-based evaluation
// 4. Better stagnation detection
func GenerateStringArtV13Optimized(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

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

	// Create enhanced importance map
	importance := createImportanceMapV13(img, edgeMap, width, height)

	// Pre-compute all line pixels with anti-aliasing
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)

	fmt.Printf("=== String Art Generator v13.0 Optimized ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Adaptive parameters
	baseWeight := float64(config.LineWeight)
	recentScores := make([]float64, 0, 25)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		// Improved adaptive line weight with better calibration for mobile rendering
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.45*progress) // Slightly more aggressive reduction
		if adaptiveWeight < 12 {
			adaptiveWeight = 12
		}

		bestLine := findBestLineV13(currentPin, pins, canvas, target, edgeMap, importance,
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

		// Apply line to canvas with calibrated alpha for mobile rendering match
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				// Calibrated source-over compositing to better match mobile SVG rendering
				alpha := (adaptiveWeight / 255.0) * 0.85 // Calibrated factor
				canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To
		usedLines[key]++

		if (i+1)%200 == 0 {
			mse := calculateMSEV13(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, adaptiveWeight)
		}
	}

	// Enhanced Line Removal Pass with SSIM-based evaluation
	fmt.Println("\n--- Enhanced Line Removal Pass ---")
	beforeSSIM := calculateSSIMV13(canvas, target, width, height)
	fmt.Printf("Before removal: SSIM = %.4f\n", beforeSSIM)

	removedCount := 0
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		key := [2]int{min(line.From, line.To), max(line.From, line.To)}
		pixels := linePixels[key]

		// Temporarily remove line
		for _, pixel := range pixels {
			if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
				alpha := (baseWeight / 255.0) * 0.85
				if 1.0-alpha*pixel.Weight > 0.001 {
					canvas[pixel.Y][pixel.X] = (canvas[pixel.Y][pixel.X] - 0.0*alpha*pixel.Weight) / (1.0-alpha*pixel.Weight)
				}
			}
		}

		// Check if removal improves SSIM
		testSSIM := calculateSSIMV13(canvas, target, width, height)
		if testSSIM > beforeSSIM + 0.001 {
			// Keep it removed
			lines = append(lines[:i], lines[i+1:]...)
			removedCount++
			beforeSSIM = testSSIM
		} else {
			// Put it back
			for _, pixel := range pixels {
				if pixel.X >= 0 && pixel.X < width && pixel.Y >= 0 && pixel.Y < height {
					alpha := (baseWeight / 255.0) * 0.85
					canvas[pixel.Y][pixel.X] = canvas[pixel.Y][pixel.X]*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight
				}
			}
		}
	}

	afterSSIM := calculateSSIMV13(canvas, target, width, height)
	finalMSE := calculateMSEV13(canvas, target, width, height)
	fmt.Printf("After removal: SSIM = %.4f (removed %d lines)\n", afterSSIM, removedCount)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM: %.4f\n", len(lines), finalMSE, afterSSIM)

	return lines, canvas
}

// Enhanced importance map with better face detection
func createImportanceMapV13(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	centerX, centerY := float64(width)/2, float64(height)/2

	// More precise face region
	faceTop := int(float64(height) * 0.25)
	faceBottom := int(float64(height) * 0.65)
	faceLeft := int(float64(width) * 0.35)
	faceRight := int(float64(width) * 0.65)

	// Eye regions
	leftEyeX := int(float64(width) * 0.42)
	rightEyeX := int(float64(width) * 0.58)
	eyeY := int(float64(height) * 0.38)
	eyeRadius := int(float64(width) * 0.06)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Base importance from edge strength
			edgeStrength := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			importance[y][x] = 1.0 + edgeStrength*2.5

			// Distance weighting (center bias)
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			distance := math.Sqrt(dx*dx + dy*dy)
			maxDistance := math.Sqrt(centerX*centerX + centerY*centerY)
			distanceWeight := 1.0 + (1.0-distance/maxDistance)*0.6

			// Face region boost
			faceWeight := 1.0
			if x >= faceLeft && x <= faceRight && y >= faceTop && y <= faceBottom {
				faceWeight = 1.8
			}

			// Eye region boost
			eyeWeight := 1.0
			leftEyeDist := math.Sqrt(float64((x-leftEyeX)*(x-leftEyeX) + (y-eyeY)*(y-eyeY)))
			rightEyeDist := math.Sqrt(float64((x-rightEyeX)*(x-rightEyeX) + (y-eyeY)*(y-eyeY)))
			if leftEyeDist < float64(eyeRadius) || rightEyeDist < float64(eyeRadius) {
				eyeWeight = 2.5
			}

			// Dark feature boost (eyes, nose, mouth)
			grayValue := float64(img.GrayAt(x, y).Y)
			contrastWeight := 1.0
			if grayValue < 90 {
				contrastWeight = 2.0
			} else if grayValue > 170 {
				contrastWeight = 1.2
			}

			importance[y][x] *= distanceWeight * faceWeight * eyeWeight * contrastWeight
		}
	}

	return importance
}

// Enhanced line finding with better scoring
func findBestLineV13(currentPin int, pins []Pin, canvas [][]float64, target [][]float64, edgeMap *image.Gray, importance [][]float64,
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

				score := calculateLineScoreV13(pixels, canvas, target, importance, lineWeight)

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
			bestPin = candidate.toPin
		}
	}

	return Line{From: currentPin, To: bestPin, Score: bestScore}
}

// Enhanced line scoring
func calculateLineScoreV13(pixels []AntiAliasedPixel, canvas [][]float64, target [][]float64, importance [][]float64, lineWeight float64) float64 {
	totalScore := 0.0
	totalWeight := 0.0
	alpha := (lineWeight / 255.0) * 0.85

	for _, pixel := range pixels {
		x, y := pixel.X, pixel.Y
		if x >= 0 && x < len(canvas[0]) && y >= 0 && y < len(canvas) {
			currentVal := canvas[y][x]
			targetVal := target[y][x]
			newVal := currentVal*(1.0-alpha*pixel.Weight) + 0.0*alpha*pixel.Weight

			// Error reduction
			currentError := (currentVal - targetVal) * (currentVal - targetVal)
			newError := (newVal - targetVal) * (newVal - targetVal)
			improvement := currentError - newError

			// Weight by importance and pixel weight
			pixelImportance := importance[y][x]
			score := improvement * pixelImportance * pixel.Weight

			totalScore += score
			totalWeight += pixelImportance * pixel.Weight
		}
	}

	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	return 0
}

// MSE calculation
func calculateMSEV13(canvas [][]float64, target [][]float64, width, height int) float64 {
	var mse float64
	count := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(canvas) && x < len(canvas[y]) && y < len(target) && x < len(target[y]) {
				diff := canvas[y][x] - target[y][x]
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

// SSIM calculation for line removal evaluation
func calculateSSIMV13(canvas [][]float64, target [][]float64, width, height int) float64 {
	var meanCanvas, meanTarget, varCanvas, varTarget, covar float64
	count := 0

	// Calculate means
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(canvas) && x < len(canvas[y]) && y < len(target) && x < len(target[y]) {
				meanCanvas += canvas[y][x]
				meanTarget += target[y][x]
				count++
			}
		}
	}
	meanCanvas /= float64(count)
	meanTarget /= float64(count)

	// Calculate variances and covariance
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y < len(canvas) && x < len(canvas[y]) && y < len(target) && x < len(target[y]) {
				canvasVal := canvas[y][x]
				targetVal := target[y][x]
				
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