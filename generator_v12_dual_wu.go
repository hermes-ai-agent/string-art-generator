package main

import (
	"fmt"
	"image"
	"math"
	"sync"
)

// GenerateStringArtV12DualWu generates dual-color string art (black + white threads)
// using Wu's anti-aliased line drawing for both passes.
//
// Key improvements over legacy dual-color:
// 1. Wu's AA lines (matches SVG renderer) instead of Bresenham
// 2. Separate canvas for black and white passes
// 3. Interleaved optimization: alternate black/white to minimize residual error
// 4. Source-over compositing with per-pixel coverage
// 5. Multi-start greedy + iterative replacement for both passes
func GenerateStringArtV12DualWu(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, []Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v12.0 (Dual-Color Wu AA) ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Black lines: %d, White lines: %d\n", config.NumLines, config.NumLines)

	alpha := 0.20
	if config.Opacity > 0 && config.Opacity < 1.0 {
		alpha = config.Opacity
	}
	fmt.Printf("Alpha: %.3f\n", alpha)

	// Create importance map
	importance := createImportanceMapV10Enhanced(img, edgeMap, width, height)

	// Pre-compute line pixels using Wu's AA
	fmt.Println("Pre-computing line pixels (Wu AA)...")
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := centerX - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)
	linePixelsWu := precomputeLinePixelsWu(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixelsWu))

	// Prepare target (normalized 0.0-1.0, 1.0=white, 0.0=black)
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y) / 255.0
		}
	}

	// Canvas starts white (1.0)
	canvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 1.0
		}
	}

	// PASS 1: Black threads (darken areas that are too bright)
	fmt.Println("\n=== PASS 1: Black Threads ===")
	blackLines := dualPassWu(pins, canvas, target, importance, linePixelsWu, config, width, height, alpha, true)

	// Apply black lines to canvas
	applyLinesWuDual(blackLines, canvas, linePixelsWu, alpha, true)
	blackMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("After black pass: %d lines, MSE: %.1f\n", len(blackLines), blackMSE)

	// PASS 2: White threads (lighten areas that are too dark)
	fmt.Println("\n=== PASS 2: White Threads ===")
	whiteLines := dualPassWu(pins, canvas, target, importance, linePixelsWu, config, width, height, alpha, false)

	// Apply white lines to canvas
	applyLinesWuDual(whiteLines, canvas, linePixelsWu, alpha, false)
	whiteMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("After white pass: %d lines, MSE: %.1f\n", len(whiteLines), whiteMSE)

	// PASS 3: Interleaved refinement (alternate black/white)
	fmt.Println("\n=== PASS 3: Interleaved Refinement ===")
	for round := 0; round < 2; round++ {
		extraBlack := dualRefinementPass(pins, canvas, target, importance, linePixelsWu, config, width, height, alpha, true, 500)
		applyLinesWuDual(extraBlack, canvas, linePixelsWu, alpha, true)
		blackLines = append(blackLines, extraBlack...)

		extraWhite := dualRefinementPass(pins, canvas, target, importance, linePixelsWu, config, width, height, alpha, false, 500)
		applyLinesWuDual(extraWhite, canvas, linePixelsWu, alpha, false)
		whiteLines = append(whiteLines, extraWhite...)

		finalMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
		fmt.Printf("Round %d: black=%d, white=%d, MSE=%.1f\n", round+1, len(blackLines), len(whiteLines), finalMSE)
	}

	fmt.Printf("\nFinal: %d black + %d white = %d total lines\n", len(blackLines), len(whiteLines), len(blackLines)+len(whiteLines))

	return blackLines, whiteLines, canvas
}

// dualPassWu runs a greedy pass for one color (black or white)
func dualPassWu(pins []Pin, canvas, target, importance [][]float64,
	linePixelsWu map[[2]int][]WuPixel, config *Config,
	width, height int, alpha float64, isBlack bool) []Line {

	colorName := "white"
	if isBlack {
		colorName = "black"
	}

	bestStartPins := findBestStartPinsDual(target, canvas, pins, width, height, 3, isBlack)
	fmt.Printf("Best starting pins for %s: %v\n", colorName, bestStartPins)

	var bestLines []Line
	bestMSE := math.MaxFloat64

	for attempt, startPin := range bestStartPins {
		canvasCopy := copyCanvasFloat(canvas, width, height)
		lines := greedyPassDualWu(startPin, pins, canvasCopy, target, importance,
			linePixelsWu, config, width, height, alpha, isBlack)
		mse := calculateMSENorm(canvasCopy, target, width, height) * 255 * 255
		fmt.Printf("  Attempt %d (%s): %d lines, MSE: %.1f\n", attempt+1, colorName, len(lines), mse)
		if mse < bestMSE {
			bestMSE = mse
			bestLines = lines
		}
	}

	return bestLines
}

// greedyPassDualWu greedy line selection for one color pass
func greedyPassDualWu(startPin int, pins []Pin, canvas, target, importance [][]float64,
	linePixelsWu map[[2]int][]WuPixel, config *Config,
	width, height int, alpha float64, isBlack bool) []Line {

	var lines []Line
	currentPin := startPin
	lastPins := make([]int, 0, 20)
	lineCount := make(map[[2]int]int)
	numPins := len(pins)

	for len(lines) < config.NumLines {
		bestScore := -math.MaxFloat64
		bestPin := -1

		type candidate struct {
			pin   int
			score float64
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		candidates := make([]candidate, 0, numPins)

		pinBatch := make([]int, 0, numPins)
		for p := 0; p < numPins; p++ {
			if p == currentPin {
				continue
			}
			dist := p - currentPin
			if dist < 0 {
				dist = -dist
			}
			if dist > numPins/2 {
				dist = numPins - dist
			}
			if dist < config.MinDistance {
				continue
			}
			recent := false
			for _, lp := range lastPins {
				if lp == p {
					recent = true
					break
				}
			}
			if recent {
				continue
			}
			key := [2]int{min2(currentPin, p), max2(currentPin, p)}
			if lineCount[key] >= 3 {
				continue
			}
			pinBatch = append(pinBatch, p)
		}

		batchSize := len(pinBatch) / config.Workers
		if batchSize == 0 {
			batchSize = 1
		}

		for i := 0; i < len(pinBatch); i += batchSize {
			end := i + batchSize
			if end > len(pinBatch) {
				end = len(pinBatch)
			}
			batch := pinBatch[i:end]
			wg.Add(1)
			go func(batch []int) {
				defer wg.Done()
				for _, p := range batch {
					key := [2]int{min2(currentPin, p), max2(currentPin, p)}
					pixels, ok := linePixelsWu[key]
					if !ok {
						continue
					}
					score := scoreDualWu(pixels, canvas, target, importance, alpha, isBlack)
					mu.Lock()
					candidates = append(candidates, candidate{p, score})
					mu.Unlock()
				}
			}(batch)
		}
		wg.Wait()

		for _, c := range candidates {
			if c.score > bestScore {
				bestScore = c.score
				bestPin = c.pin
			}
		}

		if bestPin == -1 || bestScore <= 0 {
			break
		}

		key := [2]int{min2(currentPin, bestPin), max2(currentPin, bestPin)}
		pixels := linePixelsWu[key]
		drawWuDual(pixels, canvas, alpha, isBlack)
		lineCount[key]++

		lines = append(lines, Line{From: currentPin, To: bestPin})

		lastPins = append(lastPins, currentPin)
		if len(lastPins) > 20 {
			lastPins = lastPins[1:]
		}
		currentPin = bestPin
	}

	return lines
}

// scoreDualWu scores a line for dual-color mode
func scoreDualWu(pixels []WuPixel, canvas, target, importance [][]float64, alpha float64, isBlack bool) float64 {
	score := 0.0
	for _, px := range pixels {
		if px.X < 0 || px.Y < 0 || px.X >= len(canvas[0]) || px.Y >= len(canvas) {
			continue
		}
		current := canvas[px.Y][px.X]
		tgt := target[px.Y][px.X]
		imp := importance[px.Y][px.X]

		var newVal float64
		if isBlack {
			newVal = current * (1.0 - alpha*px.Coverage)
		} else {
			newVal = current + (1.0-current)*(alpha*px.Coverage)
		}

		oldErr := (current - tgt) * (current - tgt)
		newErr := (newVal - tgt) * (newVal - tgt)
		improvement := (oldErr - newErr) * imp

		if improvement > 0 {
			score += improvement
		}
	}
	return score
}

// drawWuDual applies a Wu AA line to canvas for dual-color
func drawWuDual(pixels []WuPixel, canvas [][]float64, alpha float64, isBlack bool) {
	for _, px := range pixels {
		if px.X < 0 || px.Y < 0 || px.X >= len(canvas[0]) || px.Y >= len(canvas) {
			continue
		}
		if isBlack {
			canvas[px.Y][px.X] *= (1.0 - alpha*px.Coverage)
		} else {
			canvas[px.Y][px.X] += (1.0 - canvas[px.Y][px.X]) * (alpha * px.Coverage)
		}
		if canvas[px.Y][px.X] < 0 {
			canvas[px.Y][px.X] = 0
		}
		if canvas[px.Y][px.X] > 1 {
			canvas[px.Y][px.X] = 1
		}
	}
}

// applyLinesWuDual applies a set of lines to canvas
func applyLinesWuDual(lines []Line, canvas [][]float64, linePixelsWu map[[2]int][]WuPixel, alpha float64, isBlack bool) {
	for _, line := range lines {
		key := [2]int{min2(line.From, line.To), max2(line.From, line.To)}
		pixels, ok := linePixelsWu[key]
		if !ok {
			continue
		}
		drawWuDual(pixels, canvas, alpha, isBlack)
	}
}

// dualRefinementPass adds more lines of one color to refine
func dualRefinementPass(pins []Pin, canvas, target, importance [][]float64,
	linePixelsWu map[[2]int][]WuPixel, config *Config,
	width, height int, alpha float64, isBlack bool, maxLines int) []Line {

	cfgCopy := *config
	cfgCopy.NumLines = maxLines
	return greedyPassDualWu(0, pins, canvas, target, importance,
		linePixelsWu, &cfgCopy, width, height, alpha, isBlack)
}

// findBestStartPinsDual finds best starting pins for dual-color pass
func findBestStartPinsDual(target, canvas [][]float64, pins []Pin, width, height, n int, isBlack bool) []int {
	type pinScore struct {
		pin   int
		score float64
	}

	scores := make([]pinScore, len(pins))
	for i, pin := range pins {
		px := int(pin.X)
		py := int(pin.Y)
		if px < 0 || py < 0 || px >= width || py >= height {
			continue
		}
		tgt := target[py][px]
		cur := canvas[py][px]
		if isBlack {
			scores[i] = pinScore{i, cur - tgt}
		} else {
			scores[i] = pinScore{i, tgt - cur}
		}
	}

	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	result := make([]int, 0, n)
	for i := 0; i < n && i < len(scores); i++ {
		result = append(result, scores[i].pin)
	}
	return result
}

// copyCanvasFloat creates a deep copy of float canvas
func copyCanvasFloat(canvas [][]float64, width, height int) [][]float64 {
	c := make([][]float64, height)
	for y := 0; y < height; y++ {
		c[y] = make([]float64, width)
		copy(c[y], canvas[y])
	}
	return c
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}
