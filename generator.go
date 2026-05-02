package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArt generates string art lines using greedy algorithm with parallel evaluation
// Returns lines and final canvas state
func GenerateStringArt(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]int) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	// Create canvas (inverted: 255 = white, 0 = black)
	canvas := make([][]int, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]int, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 255
		}
	}

	// Generate pins around circle
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := math.Min(centerX, centerY) - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	// Track generated lines
	lines := make([]Line, 0, config.NumLines)
	currentPin := 0

	// v2.2.0: Adaptive stopping - track recent scores
	recentScores := make([]float64, 0, 10)

	// Generate lines
	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLine(currentPin, pins, canvas, img, edgeMap, config)
		
		if bestLine.Score <= 0 {
			break // No more improvement possible
		}

		// v2.2.0: Adaptive stopping - check quality plateau
		if config.AdaptiveStop && len(recentScores) >= 10 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			
			if avgScore < config.StopThreshold {
				fmt.Printf("Adaptive stop at line %d (avg score: %.2f < threshold: %.2f)\n", 
					i, avgScore, config.StopThreshold)
				break
			}
		}

		// v2.1.0: Apply opacity to line weight
		effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
		
		// Draw line on canvas
		drawLine(canvas, pins[bestLine.From], pins[bestLine.To], effectiveWeight)
		
		lines = append(lines, bestLine)
		currentPin = bestLine.To

		// v2.2.0: Track score for adaptive stopping
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 10 {
			recentScores = recentScores[1:]
		}

		// Progress reporting
		if (i+1)%100 == 0 {
			fmt.Printf("Progress: %d/%d lines (score: %.2f)\n", i+1, config.NumLines, bestLine.Score)
		}
	}

	fmt.Printf("Generated %d lines\n", len(lines))
	return lines, canvas
}

// findBestLine finds the best next line using parallel evaluation
func findBestLine(fromPin int, pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, config *Config) Line {
	numPins := len(pins)
	
	// v2.1.0: Random sampling optimization
	candidatePins := make([]int, 0, numPins)
	for toPin := 0; toPin < numPins; toPin++ {
		// Skip invalid connections
		distance := abs(toPin - fromPin)
		if distance < config.MinDistance && distance > 0 {
			continue
		}
		if numPins-distance < config.MinDistance {
			continue
		}
		if toPin == fromPin {
			continue
		}
		candidatePins = append(candidatePins, toPin)
	}

	// v2.1.0: Apply random sampling if enabled
	if config.RandomSampling && len(candidatePins) > config.SampleSize {
		// Shuffle and take first SampleSize elements
		for i := range candidatePins {
			j := i + int(math.Floor(float64(len(candidatePins)-i)*0.5)) // Simple pseudo-random
			candidatePins[i], candidatePins[j] = candidatePins[j], candidatePins[i]
		}
		candidatePins = candidatePins[:config.SampleSize]
	}
	
	// Create worker pool
	type job struct {
		toPin int
	}
	type result struct {
		toPin int
		score float64
	}

	jobs := make(chan job, len(candidatePins))
	results := make(chan result, len(candidatePins))
	
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < config.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				score := evaluateLine(fromPin, j.toPin, pins, canvas, img, edgeMap, config)
				
				// v2.2.0: Look-ahead optimization
				if config.LookAhead {
					futureScore := evaluateLookAhead(j.toPin, pins, canvas, img, edgeMap, config)
					score = score*0.7 + futureScore*0.3 // Weighted combination
				}
				
				results <- result{toPin: j.toPin, score: score}
			}
		}()
	}

	// Send jobs
	go func() {
		for _, toPin := range candidatePins {
			jobs <- job{toPin: toPin}
		}
		close(jobs)
	}()

	// Close results when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and find best
	bestScore := 0.0
	bestToPin := -1

	for r := range results {
		if r.score > bestScore {
			bestScore = r.score
			bestToPin = r.toPin
		}
	}

	return Line{
		From:  fromPin,
		To:    bestToPin,
		Score: bestScore,
	}
}

// evaluateLine calculates the score for a potential line
// Python v2.0.0 logic: score = remaining darkness in target image
func evaluateLine(fromPin, toPin int, pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, config *Config) float64 {
	from := pins[fromPin]
	to := pins[toPin]

	x0 := int(from.X)
	y0 := int(from.Y)
	x1 := int(to.X)
	y1 := int(to.Y)

	pixels := GetPixelsOnLine(x0, y0, x1, y1)
	
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	score := 0.0
	validPixels := 0

	for _, p := range pixels {
		x, y := p[0], p[1]
		
		// Check bounds
		if x < 0 || x >= width || y < 0 || y >= height {
			continue
		}

		// Python approach:
		// - img_array starts as original image (dark = low value)
		// - After each line, img_array gets LIGHTENED (value increases)
		// - Score = darkness = 255 - img_array (remaining darkness to cover)
		//
		// Go equivalent:
		// - img = original image (dark = low value) - IMMUTABLE
		// - canvas starts WHITE (255), gets DARKENED (value decreases)
		// - Remaining work = target_darkness - canvas_darkness
		// - target_darkness = 255 - img
		// - canvas_darkness = 255 - canvas
		// - remaining = (255 - img) - (255 - canvas) = canvas - img
		
		targetDarkness := 255 - int(img.GrayAt(x, y).Y)
		canvasDarkness := 255 - canvas[y][x]
		remainingDarkness := targetDarkness - canvasDarkness
		
		// Only score if there's remaining darkness to add
		if remainingDarkness <= 0 {
			continue
		}
		
		// Get edge strength from original edge map
		edgeStrength := float64(edgeMap.GrayAt(x, y).Y)

		// Combined score with edge weighting
		pixelScore := float64(remainingDarkness) + (edgeStrength * config.EdgeWeight)
		
		score += pixelScore
		validPixels++
	}

	// Normalize by line length
	if validPixels > 0 {
		score /= float64(validPixels)
	}

	return score
}

// evaluateLookAhead calculates future score for look-ahead optimization (v2.2.0)
func evaluateLookAhead(fromPin int, pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, config *Config) float64 {
	// Sample a few future moves and return average score
	numPins := len(pins)
	sampleSize := min(20, numPins/10) // Sample 20 pins or 10% of total
	
	totalScore := 0.0
	validSamples := 0
	
	for i := 0; i < sampleSize; i++ {
		toPin := (fromPin + i*numPins/sampleSize) % numPins
		
		// Skip invalid connections
		distance := abs(toPin - fromPin)
		if distance < config.MinDistance && distance > 0 {
			continue
		}
		if numPins-distance < config.MinDistance {
			continue
		}
		
		score := evaluateLine(fromPin, toPin, pins, canvas, img, edgeMap, config)
		totalScore += score
		validSamples++
	}
	
	if validSamples > 0 {
		return totalScore / float64(validSamples)
	}
	return 0.0
}

// drawLine draws a line on the canvas
func drawLine(canvas [][]int, from, to Pin, weight int) {
	x0 := int(from.X)
	y0 := int(from.Y)
	x1 := int(to.X)
	y1 := int(to.Y)

	pixels := GetPixelsOnLine(x0, y0, x1, y1)
	
	height := len(canvas)
	width := len(canvas[0])

	for _, p := range pixels {
		x, y := p[0], p[1]
		
		if x >= 0 && x < width && y >= 0 && y < height {
			// Darken the canvas (subtract weight, but don't go below 0)
			canvas[y][x] = max(0, canvas[y][x]-weight)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
