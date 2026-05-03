package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV11 implements string art with Wu's anti-aliased line drawing
// to better match SVG rendering. The key insight is that SVG renderers use
// anti-aliased lines, not Bresenham. By matching the rasterization method,
// the canvas simulation more accurately predicts the SVG output.
//
// Changes from V10:
// 1. Wu's anti-aliased lines instead of Bresenham (matches SVG renderer)
// 2. Each pixel has fractional coverage (0.0-1.0) instead of binary
// 3. Source-over compositing uses per-pixel coverage: canvas *= (1 - alpha * coverage)
// 4. Multi-start with iterative replacement from V10.1
// 5. Calibrated alpha to match SVG stroke-width 0.18mm at 800px with anti-aliasing
func GenerateStringArtV11(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v11.0 (Wu Anti-Aliased) ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	// Alpha calibrated for Wu's AA lines matching SVG render
	// Wu's lines spread alpha across 2 pixels per step, so effective alpha per pixel is lower
	// Calibrate: SVG stroke 0.18mm at 800px ≈ 0.24px wide
	// Wu's AA distributes this as ~0.76 coverage on main pixel + ~0.24 on neighbor
	// Effective alpha = base_alpha * coverage
	alpha := 0.20 // Calibrated: Wu AA with this alpha gives best SVG SSIM
	if config.Opacity > 0 && config.Opacity < 1.0 {
		alpha = config.Opacity // Allow override via --opacity flag
	}
	fmt.Printf("Base alpha: %.3f (Wu AA distributes per-pixel)\n", alpha)

	// Create importance map
	importance := createImportanceMapV10Enhanced(img, edgeMap, width, height)

	// Pre-compute line pixels using Wu's anti-aliased algorithm
	fmt.Println("Pre-computing line pixels (Wu anti-aliased)...")
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := centerX - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	linePixelsWu := precomputeLinePixelsWu(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments (Wu AA)\n", len(linePixelsWu))

	// Prepare target
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y) / 255.0
		}
	}

	// Find best starting pins
	bestStartPins := findBestStartPins(target, pins, width, height, 3)
	fmt.Printf("Best starting pins: %v\n", bestStartPins)

	var bestLines []Line
	var bestCanvas [][]float64
	bestMSE := math.MaxFloat64

	for attempt, startPin := range bestStartPins {
		fmt.Printf("\n--- Attempt %d: Starting from pin %d ---\n", attempt+1, startPin)

		canvas := make([][]float64, height)
		for y := 0; y < height; y++ {
			canvas[y] = make([]float64, width)
			for x := 0; x < width; x++ {
				canvas[y][x] = 1.0
			}
		}

		lines := greedyPhaseWu(startPin, pins, canvas, target, importance,
			linePixelsWu, config, width, height, alpha)

		// Line removal
		lines = lineRemovalWu(lines, canvas, target, importance, linePixelsWu, config, width, height, alpha)

		mse := calculateMSENorm(canvas, target, width, height) * 255 * 255
		fmt.Printf("Attempt %d result: %d lines, MSE: %.1f\n", attempt+1, len(lines), mse)

		if mse < bestMSE {
			bestMSE = mse
			bestLines = lines
			bestCanvas = canvas
		}
	}

	fmt.Printf("\nBest attempt MSE: %.1f with %d lines\n", bestMSE, len(bestLines))

	// Phase 3: Iterative Line Replacement
	fmt.Println("\n--- Phase 3: Iterative Line Replacement ---")
	bestLines, bestCanvas = iterativeReplacementWu(bestLines, bestCanvas, target, importance,
		linePixelsWu, pins, config, width, height, alpha)

	// Convert canvas to [0,255]
	finalCanvas := make([][]float64, height)
	for y := 0; y < height; y++ {
		finalCanvas[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			finalCanvas[y][x] = bestCanvas[y][x] * 255.0
		}
	}

	finalMSE := calculateMSENorm(bestCanvas, target, width, height) * 255 * 255
	fmt.Printf("\nFinal: %d lines, MSE: %.1f\n", len(bestLines), finalMSE)

	return bestLines, finalCanvas
}

// WuPixel represents a pixel with fractional coverage from Wu's algorithm
type WuPixel struct {
	X, Y     int
	Coverage float64 // 0.0 to 1.0
}

// precomputeLinePixelsWu precomputes line pixels using Wu's anti-aliased algorithm
func precomputeLinePixelsWu(pins []Pin, width, height, minDistance int) map[[2]int][]WuPixel {
	linePixels := make(map[[2]int][]WuPixel)
	numPins := len(pins)

	for from := 0; from < numPins; from++ {
		for to := from + 1; to < numPins; to++ {
			distance := to - from
			if distance < minDistance || numPins-distance < minDistance {
				continue
			}

			key := [2]int{from, to}
			pixels := wuLine(pins[from].X, pins[from].Y, pins[to].X, pins[to].Y, width, height)
			if len(pixels) > 0 {
				linePixels[key] = pixels
			}
		}
	}

	return linePixels
}

// wuLine implements Wu's anti-aliased line drawing algorithm
func wuLine(x0f, y0f, x1f, y1f float64, width, height int) []WuPixel {
	var pixels []WuPixel

	steep := math.Abs(y1f-y0f) > math.Abs(x1f-x0f)

	if steep {
		x0f, y0f = y0f, x0f
		x1f, y1f = y1f, x1f
	}

	if x0f > x1f {
		x0f, x1f = x1f, x0f
		y0f, y1f = y1f, y0f
	}

	dx := x1f - x0f
	dy := y1f - y0f

	var gradient float64
	if dx == 0 {
		gradient = 1.0
	} else {
		gradient = dy / dx
	}

	// Handle first endpoint
	xend := math.Round(x0f)
	yend := y0f + gradient*(xend-x0f)
	xpxl1 := int(xend)
	ypxl1 := int(math.Floor(yend))

	// Handle second endpoint
	xend2 := math.Round(x1f)
	xpxl2 := int(xend2)

	// Main loop
	intery := yend + gradient

	for x := xpxl1 + 1; x < xpxl2; x++ {
		y := int(math.Floor(intery))
		frac := intery - math.Floor(intery)

		if steep {
			if y >= 0 && y < width && x >= 0 && x < height {
				pixels = append(pixels, WuPixel{X: y, Y: x, Coverage: 1.0 - frac})
			}
			if y+1 >= 0 && y+1 < width && x >= 0 && x < height {
				pixels = append(pixels, WuPixel{X: y + 1, Y: x, Coverage: frac})
			}
		} else {
			if x >= 0 && x < width && y >= 0 && y < height {
				pixels = append(pixels, WuPixel{X: x, Y: y, Coverage: 1.0 - frac})
			}
			if x >= 0 && x < width && y+1 >= 0 && y+1 < height {
				pixels = append(pixels, WuPixel{X: x, Y: y + 1, Coverage: frac})
			}
		}

		intery += gradient
	}

	// Add endpoints
	if steep {
		if ypxl1 >= 0 && ypxl1 < width && xpxl1 >= 0 && xpxl1 < height {
			pixels = append(pixels, WuPixel{X: ypxl1, Y: xpxl1, Coverage: 1.0})
		}
		y2 := int(math.Floor(y0f + gradient*(float64(xpxl2)-x0f)))
		if y2 >= 0 && y2 < width && xpxl2 >= 0 && xpxl2 < height {
			pixels = append(pixels, WuPixel{X: y2, Y: xpxl2, Coverage: 1.0})
		}
	} else {
		if xpxl1 >= 0 && xpxl1 < width && ypxl1 >= 0 && ypxl1 < height {
			pixels = append(pixels, WuPixel{X: xpxl1, Y: ypxl1, Coverage: 1.0})
		}
		y2 := int(math.Floor(y0f + gradient*(float64(xpxl2)-x0f)))
		if xpxl2 >= 0 && xpxl2 < width && y2 >= 0 && y2 < height {
			pixels = append(pixels, WuPixel{X: xpxl2, Y: y2, Coverage: 1.0})
		}
	}

	return pixels
}

// greedyPhaseWu is the greedy line selection using Wu AA pixels
func greedyPhaseWu(startPin int, pins []Pin, canvas, target, importance [][]float64,
	linePixelsWu map[[2]int][]WuPixel, config *Config, width, height int, alpha float64) []Line {

	lines := make([]Line, 0, config.NumLines)
	currentPin := startPin
	usedLines := make(map[[2]int]int)

	fmt.Println("Phase 1: Greedy Add (Wu AA)...")
	recentScores := make([]float64, 0, 100)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineWu(currentPin, pins, canvas, target, importance,
			linePixelsWu, config, usedLines, width, height, alpha)

		if bestLine.Score <= 0.0001 {
			if stagnationCount < 5 {
				stagnationCount++
				currentPin = findHighErrorPin(canvas, target, importance, pins, width, height)
				continue
			}
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		stagnationCount = 0

		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 100 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 100 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < 0.00005 {
				newPin := findHighErrorPin(canvas, target, importance, pins, width, height)
				if newPin != currentPin {
					currentPin = newPin
					recentScores = recentScores[:0]
					continue
				}
				fmt.Printf("Stopping at line %d (quality plateau)\n", i)
				break
			}
		}

		// Apply line using Wu AA coverage
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixelsWu[key]
		for _, px := range pixels {
			if px.X >= 0 && px.X < width && px.Y >= 0 && px.Y < height {
				canvas[px.Y][px.X] *= (1.0 - alpha*px.Coverage)
			}
		}

		usedLines[key]++
		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%500 == 0 {
			mse := calculateMSENorm(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.6f, MSE: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse*255*255)
		}
	}

	fmt.Printf("Greedy phase: %d lines placed\n", len(lines))
	return lines
}

// findBestLineWu finds the best line using Wu AA evaluation
func findBestLineWu(fromPin int, pins []Pin, canvas, target [][]float64,
	importance [][]float64, linePixelsWu map[[2]int][]WuPixel, config *Config, usedLines map[[2]int]int,
	width, height int, alpha float64) Line {

	numPins := len(pins)

	type candidate struct {
		toPin int
		score float64
	}

	candidates := make([]candidate, 0, numPins)
	var mu sync.Mutex
	var wg sync.WaitGroup

	type evalTask struct {
		toPin int
		key   [2]int
	}

	tasks := make([]evalTask, 0, numPins)
	for toPin := 0; toPin < numPins; toPin++ {
		if toPin == fromPin {
			continue
		}
		key := [2]int{min(fromPin, toPin), max(fromPin, toPin)}

		distance := key[1] - key[0]
		if distance < config.MinDistance || numPins-distance < config.MinDistance {
			continue
		}

		// Adaptive reuse limit
		maxReuse := 4
		if usedLines[key] >= maxReuse {
			continue
		}

		tasks = append(tasks, evalTask{toPin: toPin, key: key})
	}

	batchSize := (len(tasks) + config.Workers - 1) / config.Workers

	for w := 0; w < config.Workers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > len(tasks) {
			end = len(tasks)
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			localCandidates := make([]candidate, 0, end-start)

			for ti := start; ti < end; ti++ {
				task := tasks[ti]

				score := evaluateLineWu(task.key, canvas, target, importance,
					linePixelsWu, width, height, alpha)

				if usedLines[task.key] > 0 {
					score *= 1.0 / (1.0 + 0.5*float64(usedLines[task.key]))
				}

				if score > 0 {
					localCandidates = append(localCandidates, candidate{toPin: task.toPin, score: score})
				}
			}

			mu.Lock()
			candidates = append(candidates, localCandidates...)
			mu.Unlock()
		}(start, end)
	}

	wg.Wait()

	if len(candidates) == 0 {
		return Line{From: fromPin, To: -1, Score: 0}
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return Line{From: fromPin, To: best.toPin, Score: best.score}
}

// evaluateLineWu evaluates a line using Wu AA coverage-weighted scoring
func evaluateLineWu(key [2]int, canvas, target [][]float64,
	importance [][]float64, linePixelsWu map[[2]int][]WuPixel,
	width, height int, alpha float64) float64 {

	pixels := linePixelsWu[key]
	if len(pixels) == 0 {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0

	for _, px := range pixels {
		x, y := px.X, px.Y
		if x < 0 || x >= width || y < 0 || y >= height {
			continue
		}

		currentVal := canvas[y][x]
		targetVal := target[y][x]

		// Wu AA source-over: new = old * (1 - alpha * coverage)
		effectiveAlpha := alpha * px.Coverage
		newVal := currentVal * (1.0 - effectiveAlpha)

		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)
		improvement := oldError - newError

		imp := importance[y][x]

		if improvement > 0 {
			totalScore += improvement * imp
			improvingPixels++
		} else if improvement < 0 {
			overDarkAmount := targetVal - newVal
			if overDarkAmount > 0.3 {
				totalScore += improvement * imp * 2.0
			} else if overDarkAmount > 0.15 {
				totalScore += improvement * imp * 1.5
			} else {
				totalScore += improvement * imp * 1.2
			}
			worseningPixels++
		}
		totalWeight += imp
	}

	if totalWeight <= 0 {
		return 0
	}

	score := totalScore / totalWeight

	totalPixels := improvingPixels + worseningPixels
	if totalPixels > 0 {
		worsenRatio := float64(worseningPixels) / float64(totalPixels)
		if worsenRatio > 0.5 {
			score *= (1.0 - worsenRatio*0.8)
		} else if worsenRatio > 0.35 {
			score *= (1.0 - worsenRatio*0.5)
		}
	}

	return score
}

// lineRemovalWu removes lines that hurt quality (Wu AA version)
func lineRemovalWu(lines []Line, canvas, target [][]float64,
	importance [][]float64, linePixelsWu map[[2]int][]WuPixel, config *Config,
	width, height int, alpha float64) []Line {

	if len(lines) == 0 {
		return lines
	}

	maxRemovals := len(lines) / 5
	beforeMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("Before removal: MSE = %.2f\n", beforeMSE)

	type removalCandidate struct {
		index       int
		improvement float64
	}

	candidates := make([]removalCandidate, 0)

	for idx, line := range lines {
		if line.From < 0 || line.To < 0 {
			continue
		}

		key := [2]int{min(line.From, line.To), max(line.From, line.To)}
		pixels := linePixelsWu[key]

		errorChange := 0.0
		for _, px := range pixels {
			x, y := px.X, px.Y
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}

			currentVal := canvas[y][x]
			targetVal := target[y][x]

			// Undo Wu AA source-over
			effectiveAlpha := alpha * px.Coverage
			newVal := currentVal / (1.0 - effectiveAlpha)
			if newVal > 1.0 {
				newVal = 1.0
			}

			oldErr := (currentVal - targetVal) * (currentVal - targetVal)
			newErr := (newVal - targetVal) * (newVal - targetVal)

			imp := importance[y][x]
			errorChange += (oldErr - newErr) * imp
		}

		if errorChange > 0 {
			candidates = append(candidates, removalCandidate{index: idx, improvement: errorChange})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].improvement > candidates[j].improvement
	})

	removedIndices := make(map[int]bool)
	removedCount := 0

	for _, c := range candidates {
		if removedCount >= maxRemovals {
			break
		}

		line := lines[c.index]
		key := [2]int{min(line.From, line.To), max(line.From, line.To)}
		pixels := linePixelsWu[key]

		for _, px := range pixels {
			x, y := px.X, px.Y
			if x >= 0 && x < width && y >= 0 && y < height {
				effectiveAlpha := alpha * px.Coverage
				canvas[y][x] /= (1.0 - effectiveAlpha)
				if canvas[y][x] > 1.0 {
					canvas[y][x] = 1.0
				}
			}
		}

		removedIndices[c.index] = true
		removedCount++
	}

	newLines := make([]Line, 0, len(lines)-removedCount)
	for idx, line := range lines {
		if !removedIndices[idx] {
			newLines = append(newLines, line)
		}
	}

	afterMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("After removal: MSE = %.2f (removed %d lines)\n", afterMSE, removedCount)

	return newLines
}

// iterativeReplacementWu tries to replace each line with a better alternative (Wu AA version)
func iterativeReplacementWu(lines []Line, canvas, target, importance [][]float64,
	linePixelsWu map[[2]int][]WuPixel, pins []Pin, config *Config,
	width, height int, alpha float64) ([]Line, [][]float64) {

	beforeMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("Before replacement: MSE = %.2f (%d lines)\n", beforeMSE, len(lines))

	replacements := 0
	maxIterations := 2

	for iter := 0; iter < maxIterations; iter++ {
		iterReplacements := 0

		for idx := range lines {
			line := lines[idx]
			if line.From < 0 || line.To < 0 {
				continue
			}

			key := [2]int{min(line.From, line.To), max(line.From, line.To)}
			pixels := linePixelsWu[key]

			// Remove this line temporarily
			for _, px := range pixels {
				x, y := px.X, px.Y
				if x >= 0 && x < width && y >= 0 && y < height {
					effectiveAlpha := alpha * px.Coverage
					canvas[y][x] /= (1.0 - effectiveAlpha)
					if canvas[y][x] > 1.0 {
						canvas[y][x] = 1.0
					}
				}
			}

			// Find best replacement
			bestScore := 0.0
			bestTo := -1
			numPins := len(pins)

			for toPin := 0; toPin < numPins; toPin++ {
				if toPin == line.From {
					continue
				}
				newKey := [2]int{min(line.From, toPin), max(line.From, toPin)}
				distance := newKey[1] - newKey[0]
				if distance < config.MinDistance || numPins-distance < config.MinDistance {
					continue
				}

				score := evaluateLineWu(newKey, canvas, target, importance,
					linePixelsWu, width, height, alpha)

				if score > bestScore {
					bestScore = score
					bestTo = toPin
				}
			}

			// Evaluate original
			origScore := evaluateLineWu(key, canvas, target, importance,
				linePixelsWu, width, height, alpha)

			if bestTo >= 0 && bestScore > origScore*1.1 {
				// Replace with better line
				newKey := [2]int{min(line.From, bestTo), max(line.From, bestTo)}
				newPixels := linePixelsWu[newKey]
				for _, px := range newPixels {
					x, y := px.X, px.Y
					if x >= 0 && x < width && y >= 0 && y < height {
						canvas[y][x] *= (1.0 - alpha*px.Coverage)
					}
				}
				lines[idx] = Line{From: line.From, To: bestTo, Score: bestScore}
				iterReplacements++
			} else {
				// Keep original
				for _, px := range pixels {
					x, y := px.X, px.Y
					if x >= 0 && x < width && y >= 0 && y < height {
						canvas[y][x] *= (1.0 - alpha*px.Coverage)
					}
				}
			}
		}

		replacements += iterReplacements
		mse := calculateMSENorm(canvas, target, width, height) * 255 * 255
		fmt.Printf("Replacement pass %d: %d replacements, MSE: %.2f\n", iter+1, iterReplacements, mse)

		if iterReplacements == 0 {
			break
		}
	}

	afterMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("After replacement: MSE = %.2f (total %d replacements)\n", afterMSE, replacements)

	return lines, canvas
}
