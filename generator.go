package main

import (
	"image"
	"math"
	"sync"
)

// GenerateStringArt generates string art lines using greedy algorithm with parallel evaluation
func GenerateStringArt(img *image.Gray, config *Config) []Line {
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

	// Generate lines
	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLine(currentPin, pins, canvas, img, config)
		
		if bestLine.Score <= 0 {
			break // No more improvement possible
		}

		// Draw line on canvas
		drawLine(canvas, pins[bestLine.From], pins[bestLine.To], config.LineWeight)
		
		lines = append(lines, bestLine)
		currentPin = bestLine.To
	}

	return lines
}

// findBestLine finds the best next line using parallel evaluation
func findBestLine(fromPin int, pins []Pin, canvas [][]int, img *image.Gray, config *Config) Line {
	numPins := len(pins)
	
	// Create worker pool
	type job struct {
		toPin int
	}
	type result struct {
		toPin int
		score float64
	}

	jobs := make(chan job, numPins)
	results := make(chan result, numPins)
	
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < config.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				score := evaluateLine(fromPin, j.toPin, pins, canvas, img, config)
				results <- result{toPin: j.toPin, score: score}
			}
		}()
	}

	// Send jobs
	go func() {
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
func evaluateLine(fromPin, toPin int, pins []Pin, canvas [][]int, img *image.Gray, config *Config) float64 {
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

		// Get target darkness (inverted: 0 = white, 255 = black)
		targetDarkness := 255 - int(img.GrayAt(x, y).Y)
		
		// Get current canvas darkness
		canvasDarkness := 255 - canvas[y][x]

		// Score: how much this line would improve the match
		// Only count if canvas is lighter than target (we can still darken it)
		if canvasDarkness < targetDarkness {
			improvement := float64(min(config.LineWeight, targetDarkness-canvasDarkness))
			score += improvement
			validPixels++
		}
	}

	// Normalize by line length
	if validPixels > 0 {
		score /= float64(validPixels)
	}

	return score
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
