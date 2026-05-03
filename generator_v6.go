package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV6 implements Birsak 2018-style supersampled rendering.
//
// Key insight: SVG renders thin opaque strokes at display resolution.
// A 0.18mm stroke in 600-unit viewBox at 400px display = ~0.12px wide.
// The browser anti-aliases this, creating gray levels through spatial averaging.
//
// We simulate this by:
// 1. Rendering on a high-res canvas (supersample_factor × display_res)
// 2. Drawing each line as a 1px-wide opaque black stroke
// 3. Downsampling to display resolution for scoring
// This naturally creates the same gray levels as the SVG renderer.
func GenerateStringArtV6(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	srcWidth, srcHeight := bounds.Dx(), bounds.Dy()

	// Display resolution (what the SVG will be viewed at)
	displayRes := 400 // mobile resolution - matches quality_validator.py

	// Supersample factor: 8x means we render at 3200x3200 and downsample to 400x400
	// At 3200px, a 0.18mm stroke in 600-unit viewBox = 0.96px wide ≈ 1px Bresenham line
	// This gives the most accurate simulation of SVG rendering
	ssf := 8
	hiRes := displayRes * ssf // 3200

	fmt.Printf("=== String Art Generator v6.0 (Supersampled) ===\n")
	fmt.Printf("Source: %dx%d, Display: %dx%d, HiRes: %dx%d (SSF=%d)\n",
		srcWidth, srcHeight, displayRes, displayRes, hiRes, hiRes, ssf)

	// Create target at display resolution (bilinear sampling from source)
	target := createTargetAtDisplayRes(img, displayRes, srcWidth, srcHeight)

	// NOTE: The target comes from PreprocessImage which applies gamma 1.2
	// This lightens the target. For SSIM optimization against the original image,
	// we want a darker target to push the algorithm to add more lines.
	// Darken the target by applying inverse gamma correction
	for y := 0; y < displayRes; y++ {
		for x := 0; x < displayRes; x++ {
			// Apply gamma 0.85 (darken) to counteract the preprocessing gamma 1.2
			normalized := target[y][x] / 255.0
			target[y][x] = math.Pow(normalized, 1.0/0.85) * 255.0
		}
	}

	// Create high-res canvas (starts white, uint8 for memory efficiency)
	hiCanvas := make([][]uint8, hiRes)
	for y := 0; y < hiRes; y++ {
		hiCanvas[y] = make([]uint8, hiRes)
		for x := 0; x < hiRes; x++ {
			hiCanvas[y][x] = 255
		}
	}

	// Generate pins at high-res coordinates
	hiCenterX, hiCenterY := float64(hiRes)/2, float64(hiRes)/2
	hiRadius := hiCenterX - float64(ssf)*10.0/600.0*float64(displayRes)
	hiPins := GeneratePins(config.NumPins, hiRadius, hiCenterX, hiCenterY)

	// Create importance map at display resolution
	importance := createImportanceMapV6(img, edgeMap, displayRes, srcWidth, srcHeight)

	// Create edge map at display resolution
	edgeDisplay := createEdgeDisplayMap(edgeMap, displayRes, srcWidth, srcHeight)

	// Pre-compute valid pin pairs (but NOT their pixels - too much memory)
	fmt.Println("Computing valid pin pairs...")
	validPairs := computeValidPairs(config.NumPins, config.MinDistance)
	fmt.Printf("Valid pin pairs: %d\n", len(validPairs))

	// Current downsampled canvas
	dsCanvas := downsampleCanvas(hiCanvas, ssf, displayRes)

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Min Distance: %d, Edge Weight: %.1f\n", config.MinDistance, config.EdgeWeight)

	// Track scores for stagnation detection
	recentScores := make([]float64, 0, 30)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV6(currentPin, hiPins, hiCanvas, dsCanvas, target,
			edgeDisplay, importance, validPairs, config, usedLines, ssf, displayRes, hiRes)

		if bestLine.Score <= 0.001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Stagnation detection
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 30 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 30 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < 0.01 {
				stagnationCount++
				if stagnationCount > 10 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.4f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Draw line on high-res canvas (opaque black, 1px wide Bresenham)
		drawLineOnHiCanvas(hiCanvas, hiPins[bestLine.From], hiPins[bestLine.To], hiRes)

		// Update downsampled canvas for affected region
		updateDSCanvasForLineV6(dsCanvas, hiCanvas, hiPins[bestLine.From], hiPins[bestLine.To], ssf, displayRes, hiRes)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			// Full downsample for accurate metrics
			dsCanvas = downsampleCanvas(hiCanvas, ssf, displayRes)
			mse := calculateMSEV6(dsCanvas, target, displayRes)
			ssim := quickSSIM(dsCanvas, target, displayRes)
			fmt.Printf("Progress: %d/%d lines (score: %.4f, MSE: %.1f, SSIM~%.3f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim)
		}
	}

	// Line removal pass
	fmt.Println("\n--- Line Removal Pass ---")
	lines = lineRemovalPassV6(lines, hiPins, hiCanvas, target, importance,
		ssf, displayRes, hiRes)

	// Final downsample
	dsCanvas = downsampleCanvas(hiCanvas, ssf, displayRes)
	finalMSE := calculateMSEV6(dsCanvas, target, displayRes)
	finalSSIM := quickSSIM(dsCanvas, target, displayRes)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	// Return canvas at source resolution for PNG export
	exportCanvas := make([][]float64, srcHeight)
	for y := 0; y < srcHeight; y++ {
		exportCanvas[y] = make([]float64, srcWidth)
		for x := 0; x < srcWidth; x++ {
			dy := int(float64(y) / float64(srcHeight) * float64(displayRes))
			dx := int(float64(x) / float64(srcWidth) * float64(displayRes))
			if dy >= displayRes {
				dy = displayRes - 1
			}
			if dx >= displayRes {
				dx = displayRes - 1
			}
			exportCanvas[y][x] = dsCanvas[dy][dx]
		}
	}

	return lines, exportCanvas
}

// createTargetAtDisplayRes creates target image at display resolution
func createTargetAtDisplayRes(img *image.Gray, displayRes, srcWidth, srcHeight int) [][]float64 {
	target := make([][]float64, displayRes)
	for y := 0; y < displayRes; y++ {
		target[y] = make([]float64, displayRes)
		for x := 0; x < displayRes; x++ {
			// Bilinear sampling from source
			srcXf := float64(x) / float64(displayRes) * float64(srcWidth)
			srcYf := float64(y) / float64(displayRes) * float64(srcHeight)
			srcX := int(srcXf)
			srcY := int(srcYf)
			if srcX >= srcWidth-1 {
				srcX = srcWidth - 2
			}
			if srcY >= srcHeight-1 {
				srcY = srcHeight - 2
			}
			if srcX < 0 {
				srcX = 0
			}
			if srcY < 0 {
				srcY = 0
			}
			fx := srcXf - float64(srcX)
			fy := srcYf - float64(srcY)

			v00 := float64(img.GrayAt(srcX, srcY).Y)
			v10 := float64(img.GrayAt(srcX+1, srcY).Y)
			v01 := float64(img.GrayAt(srcX, srcY+1).Y)
			v11 := float64(img.GrayAt(srcX+1, srcY+1).Y)

			target[y][x] = v00*(1-fx)*(1-fy) + v10*fx*(1-fy) + v01*(1-fx)*fy + v11*fx*fy
		}
	}
	return target
}

// createEdgeDisplayMap creates edge map at display resolution
func createEdgeDisplayMap(edgeMap *image.Gray, displayRes, srcWidth, srcHeight int) [][]float64 {
	edgeDisplay := make([][]float64, displayRes)
	for y := 0; y < displayRes; y++ {
		edgeDisplay[y] = make([]float64, displayRes)
		for x := 0; x < displayRes; x++ {
			srcX := int(float64(x) / float64(displayRes) * float64(srcWidth))
			srcY := int(float64(y) / float64(displayRes) * float64(srcHeight))
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			if srcY >= srcHeight {
				srcY = srcHeight - 1
			}
			edgeDisplay[y][x] = float64(edgeMap.GrayAt(srcX, srcY).Y) / 255.0
		}
	}
	return edgeDisplay
}

// computeValidPairs returns all valid pin pairs respecting min distance
func computeValidPairs(numPins, minDistance int) [][2]int {
	var pairs [][2]int
	for i := 0; i < numPins; i++ {
		for j := i + 1; j < numPins; j++ {
			distance := j - i
			if distance < minDistance || numPins-distance < minDistance {
				continue
			}
			pairs = append(pairs, [2]int{i, j})
		}
	}
	return pairs
}

// drawLineOnHiCanvas draws a 1px opaque black line using Bresenham
func drawLineOnHiCanvas(hiCanvas [][]uint8, from, to Pin, hiRes int) {
	x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
	y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
	x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
	y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

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
		if x >= 0 && x < hiRes && y >= 0 && y < hiRes {
			hiCanvas[y][x] = 0
		}
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
}

// updateDSCanvasForLineV6 incrementally updates downsampled canvas for a drawn line
func updateDSCanvasForLineV6(dsCanvas [][]float64, hiCanvas [][]uint8, from, to Pin, ssf, displayRes, hiRes int) {
	x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
	y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
	x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
	y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

	// Track affected display pixels
	affected := make(map[[2]int]bool)

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
		if x >= 0 && x < hiRes && y >= 0 && y < hiRes {
			dpx := x / ssf
			dpy := y / ssf
			if dpx >= displayRes {
				dpx = displayRes - 1
			}
			if dpy >= displayRes {
				dpy = displayRes - 1
			}
			affected[[2]int{dpx, dpy}] = true
		}
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

	// Recompute affected display pixels
	invSSF2 := 1.0 / float64(ssf*ssf)
	for key := range affected {
		px, py := key[0], key[1]
		sum := 0.0
		hiY := py * ssf
		hiX := px * ssf
		for dy := 0; dy < ssf; dy++ {
			for dx := 0; dx < ssf; dx++ {
				sum += float64(hiCanvas[hiY+dy][hiX+dx])
			}
		}
		dsCanvas[py][px] = sum * invSSF2
	}
}

// findBestLineV6 finds the best next line using supersampled scoring
func findBestLineV6(fromPin int, hiPins []Pin, hiCanvas [][]uint8, dsCanvas, target [][]float64,
	edgeDisplay, importance [][]float64, validPairs [][2]int,
	config *Config, usedLines map[[2]int]int, ssf, displayRes, hiRes int) Line {

	numPins := len(hiPins)

	type candidate struct {
		toPin int
		score float64
	}

	candidates := make([]candidate, 0, numPins)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Find all valid targets from current pin
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

		// Check min distance
		distance := key[1] - key[0]
		if distance < config.MinDistance || numPins-distance < config.MinDistance {
			continue
		}

		// Check usage - allow up to 5 reuses
		if usedLines[key] >= 5 {
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

				score := evaluateLineV6(hiPins[fromPin], hiPins[task.toPin],
					hiCanvas, dsCanvas, target, edgeDisplay, importance,
					config.EdgeWeight, ssf, displayRes, hiRes)

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

// evaluateLineV6 evaluates a line by simulating its effect on the downsampled canvas
// Computes line pixels on-the-fly (avoids massive memory for pre-computation)
func evaluateLineV6(from, to Pin, hiCanvas [][]uint8, dsCanvas, target [][]float64,
	edgeDisplay, importance [][]float64, edgeWeight float64, ssf, displayRes, hiRes int) float64 {

	x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
	y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
	x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
	y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

	// Count how many NEW black pixels each display pixel would get
	type pixelDelta struct {
		newBlackCount int
	}
	affected := make(map[[2]int]*pixelDelta)

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
		if x >= 0 && x < hiRes && y >= 0 && y < hiRes {
			// Only count if pixel is currently white (will change)
			if hiCanvas[y][x] != 0 {
				dpx := x / ssf
				dpy := y / ssf
				if dpx >= displayRes {
					dpx = displayRes - 1
				}
				if dpy >= displayRes {
					dpy = displayRes - 1
				}
				key := [2]int{dpx, dpy}
				info, exists := affected[key]
				if !exists {
					info = &pixelDelta{}
					affected[key] = info
				}
				info.newBlackCount++
			}
		}
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

	if len(affected) == 0 {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0
	ssfSq := float64(ssf * ssf)

	for key, info := range affected {
		px, py := key[0], key[1]
		currentVal := dsCanvas[py][px]
		targetVal := target[py][px]

		// Each new black pixel reduces the display pixel brightness by 255/ssfSq
		delta := 255.0 * float64(info.newBlackCount) / ssfSq
		newVal := currentVal - delta
		if newVal < 0 {
			newVal = 0
		}

		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)
		improvement := oldError - newError

		imp := importance[py][px]
		edgeVal := edgeDisplay[py][px]
		edgeBonus := 1.0 + edgeVal*edgeWeight*0.2

		if improvement > 0 {
			totalScore += improvement * imp * edgeBonus
			improvingPixels++
		} else if improvement < 0 {
			// Mild over-darkening penalty (1.2x)
			// With supersampled rendering, over-darkening is less visible
			// because it's distributed across sub-pixels
			totalScore += improvement * imp * 1.2
			worseningPixels++
		}
		totalWeight += imp
	}

	if totalWeight > 0 {
		score := totalScore / totalWeight

		totalPixels := improvingPixels + worseningPixels
		if totalPixels > 0 {
			worsenRatio := float64(worseningPixels) / float64(totalPixels)
			if worsenRatio > 0.4 {
				score *= (1.0 - worsenRatio)
			}
		}

		return score
	}
	return 0
}

// createImportanceMapV6 creates importance map at display resolution
func createImportanceMapV6(img *image.Gray, edgeMap *image.Gray, displayRes, srcWidth, srcHeight int) [][]float64 {
	importance := make([][]float64, displayRes)
	centerX, centerY := float64(displayRes)/2, float64(displayRes)/2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	for y := 0; y < displayRes; y++ {
		importance[y] = make([]float64, displayRes)
		for x := 0; x < displayRes; x++ {
			srcX := int(float64(x) / float64(displayRes) * float64(srcWidth))
			srcY := int(float64(y) / float64(displayRes) * float64(srcHeight))
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			if srcY >= srcHeight {
				srcY = srcHeight - 1
			}

			grayVal := float64(img.GrayAt(srcX, srcY).Y)
			edgeVal := float64(edgeMap.GrayAt(srcX, srcY).Y)

			darknessImportance := (255.0 - grayVal) / 255.0
			edgeImportance := edgeVal / 255.0

			ddx := float64(x) - centerX
			ddy := float64(y) - centerY
			dist := math.Sqrt(ddx*ddx+ddy*ddy) / maxDist
			centerWeight := math.Exp(-2.0 * dist * dist)

			base := darknessImportance*0.6 + edgeImportance*0.4
			importance[y][x] = base * (0.5 + 0.5*centerWeight)

			if importance[y][x] < 0.05 {
				importance[y][x] = 0.05
			}
		}
	}

	return importance
}

// lineRemovalPassV6 removes lines that hurt quality
func lineRemovalPassV6(lines []Line, hiPins []Pin, hiCanvas [][]uint8,
	target, importance [][]float64, ssf, displayRes, hiRes int) []Line {

	if len(lines) == 0 {
		return lines
	}

	maxRemovals := len(lines) / 10

	dsCanvas := downsampleCanvas(hiCanvas, ssf, displayRes)
	currentMSE := calculateMSEV6(dsCanvas, target, displayRes)
	fmt.Printf("Before removal: MSE = %.2f\n", currentMSE)

	type removalCandidate struct {
		index       int
		improvement float64
	}

	candidates := make([]removalCandidate, 0)
	ssfSq := float64(ssf * ssf)

	for idx, line := range lines {
		if line.From < 0 || line.To < 0 {
			continue
		}

		// Trace the line and count pixels that are currently black
		from := hiPins[line.From]
		to := hiPins[line.To]
		x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
		y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
		x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
		y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

		// Count black pixels per display block
		type blockInfo struct {
			blackCount int
		}
		blocks := make(map[[2]int]*blockInfo)

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
		errVal := dx - dy
		x, y := x0, y0

		for {
			if x >= 0 && x < hiRes && y >= 0 && y < hiRes {
				if hiCanvas[y][x] == 0 {
					dpx := x / ssf
					dpy := y / ssf
					if dpx >= displayRes {
						dpx = displayRes - 1
					}
					if dpy >= displayRes {
						dpy = displayRes - 1
					}
					key := [2]int{dpx, dpy}
					info, exists := blocks[key]
					if !exists {
						info = &blockInfo{}
						blocks[key] = info
					}
					info.blackCount++
				}
			}
			if x == x1 && y == y1 {
				break
			}
			e2 := 2 * errVal
			if e2 > -dy {
				errVal -= dy
				x += sx
			}
			if e2 < dx {
				errVal += dx
				y += sy
			}
		}

		// Calculate error change if we remove (lighten) these pixels
		errorChange := 0.0
		for key, info := range blocks {
			px, py := key[0], key[1]
			currentVal := dsCanvas[py][px]
			targetVal := target[py][px]

			// Removing makes it lighter
			delta := 255.0 * float64(info.blackCount) / ssfSq
			newVal := currentVal + delta
			if newVal > 255 {
				newVal = 255
			}

			oldErr := (currentVal - targetVal) * (currentVal - targetVal)
			newErr := (newVal - targetVal) * (newVal - targetVal)
			imp := importance[py][px]
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
		from := hiPins[line.From]
		to := hiPins[line.To]

		// Restore pixels to white
		x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
		y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
		x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
		y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

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
		errVal := dx - dy
		x, y := x0, y0

		for {
			if x >= 0 && x < hiRes && y >= 0 && y < hiRes {
				hiCanvas[y][x] = 255
			}
			if x == x1 && y == y1 {
				break
			}
			e2 := 2 * errVal
			if e2 > -dy {
				errVal -= dy
				x += sx
			}
			if e2 < dx {
				errVal += dx
				y += sy
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

	dsCanvas = downsampleCanvas(hiCanvas, ssf, displayRes)
	newMSE := calculateMSEV6(dsCanvas, target, displayRes)
	fmt.Printf("After removal: MSE = %.2f (removed %d lines)\n", newMSE, removedCount)

	return newLines
}

// downsampleCanvas downsamples high-res canvas to display resolution using box filter
func downsampleCanvas(hiCanvas [][]uint8, ssf, displayRes int) [][]float64 {
	ds := make([][]float64, displayRes)
	invSSF2 := 1.0 / float64(ssf*ssf)

	for y := 0; y < displayRes; y++ {
		ds[y] = make([]float64, displayRes)
		for x := 0; x < displayRes; x++ {
			sum := 0.0
			hiY := y * ssf
			hiX := x * ssf
			for dy := 0; dy < ssf; dy++ {
				for dx := 0; dx < ssf; dx++ {
					sum += float64(hiCanvas[hiY+dy][hiX+dx])
				}
			}
			ds[y][x] = sum * invSSF2
		}
	}

	return ds
}

// calculateMSEV6 calculates MSE between downsampled canvas and target
func calculateMSEV6(dsCanvas, target [][]float64, displayRes int) float64 {
	totalError := 0.0
	count := 0

	for y := 0; y < displayRes; y++ {
		for x := 0; x < displayRes; x++ {
			diff := dsCanvas[y][x] - target[y][x]
			totalError += diff * diff
			count++
		}
	}

	if count > 0 {
		return totalError / float64(count)
	}
	return 0
}

// quickSSIM computes a rough SSIM estimate for progress reporting
func quickSSIM(canvas, target [][]float64, size int) float64 {
	C1 := (0.01 * 255) * (0.01 * 255)
	C2 := (0.03 * 255) * (0.03 * 255)

	blockSize := 8
	numBlocks := 0
	totalSSIM := 0.0

	for by := 0; by+blockSize <= size; by += blockSize {
		for bx := 0; bx+blockSize <= size; bx += blockSize {
			var mu1, mu2 float64
			n := float64(blockSize * blockSize)

			for dy := 0; dy < blockSize; dy++ {
				for dx := 0; dx < blockSize; dx++ {
					mu1 += canvas[by+dy][bx+dx]
					mu2 += target[by+dy][bx+dx]
				}
			}
			mu1 /= n
			mu2 /= n

			var sigma1sq, sigma2sq, sigma12 float64
			for dy := 0; dy < blockSize; dy++ {
				for dx := 0; dx < blockSize; dx++ {
					d1 := canvas[by+dy][bx+dx] - mu1
					d2 := target[by+dy][bx+dx] - mu2
					sigma1sq += d1 * d1
					sigma2sq += d2 * d2
					sigma12 += d1 * d2
				}
			}
			sigma1sq /= n
			sigma2sq /= n
			sigma12 /= n

			ssim := (2*mu1*mu2 + C1) * (2*sigma12 + C2) /
				((mu1*mu1+mu2*mu2+C1)*(sigma1sq+sigma2sq+C2))
			totalSSIM += ssim
			numBlocks++
		}
	}

	if numBlocks > 0 {
		return totalSSIM / float64(numBlocks)
	}
	return 0
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
