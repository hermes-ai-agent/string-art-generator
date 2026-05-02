package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// Point represents a 2D coordinate
type Point struct {
	X, Y int
}

// bresenham returns all pixels on a line using Bresenham's algorithm
func bresenham(x0, y0, x1, y1 int) []Point {
	var pixels []Point

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	x, y := x0, y0

	for {
		pixels = append(pixels, Point{X: x, Y: y})

		if x == x1 && y == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}

	return pixels
}

// grayAtNRGBA helper for NRGBA images
func grayAtNRGBA(img *image.NRGBA, x, y int) uint8 {
	r, g, b, _ := img.At(x, y).RGBA()
	grayVal := uint8((r*299 + g*587 + b*114) / 1000 / 256)
	return grayVal
}

// GenerateStringArtDual generates dual-color string art (black + white threads)
// with transparency-aware scoring
func GenerateStringArtDual(img *image.NRGBA, edgeMap *image.Gray, config *Config) ([]Line, []Line, [][]int) {
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

	// PASS 1: Black threads (darken dark areas)
	fmt.Println("=== PASS 1: Black Threads ===")
	blackLines := generateThreadPass(pins, canvas, img, edgeMap, config, true)
	
	// Apply black lines to canvas
	for _, line := range blackLines {
		drawLineDual(canvas, pins[line.From], pins[line.To], config.LineWeight, true)
	}

	// PASS 2: White threads (lighten bright areas)
	fmt.Println("\n=== PASS 2: White Threads ===")
	whiteLines := generateThreadPass(pins, canvas, img, edgeMap, config, false)
	
	// Apply white lines to canvas
	for _, line := range whiteLines {
		drawLineDual(canvas, pins[line.From], pins[line.To], config.LineWeight, false)
	}

	fmt.Printf("\nGenerated %d black lines + %d white lines = %d total\n", 
		len(blackLines), len(whiteLines), len(blackLines)+len(whiteLines))
	
	return blackLines, whiteLines, canvas
}

// generateThreadPass generates one pass of threads (black or white)
func generateThreadPass(pins []Pin, canvas [][]int, img *image.NRGBA, edgeMap *image.Gray, config *Config, isBlack bool) []Line {
	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	recentScores := make([]float64, 0, 10)

	threadType := "white"
	if isBlack {
		threadType = "black"
	}

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineDual(currentPin, pins, canvas, img, edgeMap, config, isBlack)
		
		if bestLine.Score <= 0 {
			fmt.Printf("No more improvement for %s threads at line %d\n", threadType, i)
			break
		}

		// Adaptive stopping
		if config.AdaptiveStop && len(recentScores) >= 10 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			
			if avgScore < config.StopThreshold {
				fmt.Printf("Adaptive stop for %s threads at line %d (avg score: %.2f)\n", 
					threadType, i, avgScore)
				break
			}
		}

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		// Track score
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 10 {
			recentScores = recentScores[1:]
		}

		// Progress reporting
		if (i+1)%100 == 0 {
			fmt.Printf("Progress (%s): %d/%d lines (score: %.2f)\n", 
				threadType, i+1, config.NumLines, bestLine.Score)
		}
	}

	return lines
}

// findBestLineDual finds best line with transparency-aware scoring
func findBestLineDual(fromPin int, pins []Pin, canvas [][]int, img *image.NRGBA, edgeMap *image.Gray, config *Config, isBlack bool) Line {
	numPins := len(pins)
	
	// Get candidate pins
	candidatePins := make([]int, 0, numPins)
	for toPin := 0; toPin < numPins; toPin++ {
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

	// Random sampling if enabled
	if config.RandomSampling && len(candidatePins) > config.SampleSize {
		for i := range candidatePins {
			j := i + int(math.Floor(float64(len(candidatePins)-i)*0.5))
			candidatePins[i], candidatePins[j] = candidatePins[j], candidatePins[i]
		}
		candidatePins = candidatePins[:config.SampleSize]
	}
	
	// Parallel evaluation
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
				score := evaluateLineDual(fromPin, j.toPin, pins, canvas, img, edgeMap, config, isBlack)
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

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Find best result
	bestScore := -1e9
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

// evaluateLineDual evaluates line score with transparency awareness
func evaluateLineDual(fromPin, toPin int, pins []Pin, canvas [][]int, img *image.NRGBA, edgeMap *image.Gray, config *Config, isBlack bool) float64 {
	from := pins[fromPin]
	to := pins[toPin]
	
	// Get line pixels
	pixels := bresenham(int(from.X), int(from.Y), int(to.X), int(to.Y))
	
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	score := 0.0
	validPixels := 0
	
	effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
	
	for _, p := range pixels {
		x, y := p.X, p.Y
		
		// Bounds check
		if x < 0 || x >= width || y < 0 || y >= height {
			continue
		}
		
		// Get alpha channel
		_, _, _, alpha := img.At(x, y).RGBA()
		alphaValue := uint8(alpha >> 8)
		
		// TRANSPARENCY-AWARE: Skip transparent pixels (no scoring)
		if alphaValue < 128 {
			continue
		}
		
		validPixels++
		
		// Get current values
		currentCanvas := canvas[y][x]
		targetGray := grayAtNRGBA(img, x, y)
		edgeValue := float64(edgeMap.GrayAt(x, y).Y)
		
		// Simulate line drawing
		var newCanvas int
		if isBlack {
			// Black thread: darken (subtract)
			newCanvas = currentCanvas - effectiveWeight
			if newCanvas < 0 {
				newCanvas = 0
			}
		} else {
			// White thread: lighten (add)
			newCanvas = currentCanvas + effectiveWeight
			if newCanvas > 255 {
				newCanvas = 255
			}
		}
		
		// Calculate improvement
		oldError := float64(abs(int(targetGray) - currentCanvas))
		newError := float64(abs(int(targetGray) - newCanvas))
		improvement := oldError - newError
		
		// Edge weighting
		edgeWeight := 1.0 + (edgeValue/255.0)*config.EdgeWeight
		
		score += improvement * edgeWeight
	}
	
	// Normalize by valid pixels
	if validPixels > 0 {
		score /= float64(validPixels)
	}
	
	return score
}

// drawLineDual draws a line on canvas (black = darken, white = lighten)
func drawLineDual(canvas [][]int, from, to Pin, weight int, isBlack bool) {
	x0 := int(from.X)
	y0 := int(from.Y)
	x1 := int(to.X)
	y1 := int(to.Y)

	pixels := bresenham(x0, y0, x1, y1)
	
	height := len(canvas)
	width := len(canvas[0])

	for _, p := range pixels {
		x, y := p.X, p.Y
		
		if x >= 0 && x < width && y >= 0 && y < height {
			if isBlack {
				// Black thread: darken (subtract weight)
				canvas[y][x] = max(0, canvas[y][x]-weight)
			} else {
				// White thread: lighten (add weight)
				canvas[y][x] = min(255, canvas[y][x]+weight)
			}
		}
	}
}
