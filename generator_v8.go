package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV8 implements enhanced supersampled rendering with key improvements:
// 1. Birsak 2018 supersampled rendering (8x resolution, downsample)
// 2. Add/Remove optimization - greedy add phase followed by removal pass
// 3. Better importance map with face-region detection
// 4. Perceptual scoring using SSIM-aware metrics
// 5. Tuned source-over alpha for better canvas-SVG matching
func GenerateStringArtV8(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	srcWidth, srcHeight := bounds.Dx(), bounds.Dy()

	displayRes := 400
	ssf := 8
	hiRes := displayRes * ssf // 3200

	fmt.Printf("=== String Art Generator v8.0 (Enhanced Supersampled) ===\n")
	fmt.Printf("Source: %dx%d, Display: %dx%d, HiRes: %dx%d (SSF=%d)\n",
		srcWidth, srcHeight, displayRes, displayRes, hiRes, hiRes, ssf)

	// Create target at display resolution with improved gamma calibration
	target := createTargetV8(img, displayRes, srcWidth, srcHeight)

	// Create high-res canvas (starts white)
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

	// Create enhanced importance map with face detection
	importance := createImportanceMapV8(img, edgeMap, displayRes, srcWidth, srcHeight)

	// Create edge map at display resolution
	edgeDisplay := createEdgeDisplayMap(edgeMap, displayRes, srcWidth, srcHeight)

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

	// Phase 1: Greedy Add Phase
	fmt.Println("\n--- Phase 1: Greedy Add Phase ---")
	recentScores := make([]float64, 0, 50)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV8(currentPin, hiPins, hiCanvas, dsCanvas, target,
			edgeDisplay, importance, config, usedLines, ssf, displayRes, hiRes)

		if bestLine.Score <= 0.0001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Enhanced stagnation detection
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 50 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 50 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < 0.0005 {
				stagnationCount++
				if stagnationCount > 15 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.6f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Draw line on high-res canvas
		drawLineOnHiCanvas(hiCanvas, hiPins[bestLine.From], hiPins[bestLine.To], hiRes)

		// Update downsampled canvas efficiently
		updateDSCanvasForLineV8(dsCanvas, hiCanvas, hiPins[bestLine.From], hiPins[bestLine.To], ssf, displayRes, hiRes)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			dsCanvas = downsampleCanvas(hiCanvas, ssf, displayRes)
			mse := calculateMSEV6(dsCanvas, target, displayRes)
			ssim := quickSSIM(dsCanvas, target, displayRes)
			fmt.Printf("Progress: %d/%d lines (score: %.6f, MSE: %.1f, SSIM~%.3f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim)
		}
	}

	// Phase 2: Add/Remove Optimization
	fmt.Println("\n--- Phase 2: Add/Remove Optimization ---")
	lines = addRemoveOptimizationV8(lines, hiPins, hiCanvas, target, importance,
		edgeDisplay, config, ssf, displayRes, hiRes)

	// Final metrics
	dsCanvas = downsampleCanvas(hiCanvas, ssf, displayRes)
	finalMSE := calculateMSEV6(dsCanvas, target, displayRes)
	finalSSIM := quickSSIM(dsCanvas, target, displayRes)
	fmt.Printf("\nFinal: %d lines, MSE: %.1f, SSIM~%.3f\n", len(lines), finalMSE, finalSSIM)

	// Return canvas at source resolution
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

// createTargetV8 creates target with improved gamma calibration
func createTargetV8(img *image.Gray, displayRes, srcWidth, srcHeight int) [][]float64 {
	target := make([][]float64, displayRes)
	for y := 0; y < displayRes; y++ {
		target[y] = make([]float64, displayRes)
		for x := 0; x < displayRes; x++ {
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

			val := v00*(1-fx)*(1-fy) + v10*fx*(1-fy) + v01*(1-fx)*fy + v11*fx*fy

			// Use the same gamma calibration as v6.3 baseline (gamma 0.85)
			// This matches the successful baseline configuration
			normalized := val / 255.0
			target[y][x] = math.Pow(normalized, 1.0/0.85) * 255.0
		}
	}
	return target
}

// createImportanceMapV8 creates enhanced importance map with face-region detection
func createImportanceMapV8(img *image.Gray, edgeMap *image.Gray, displayRes, srcWidth, srcHeight int) [][]float64 {
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

			// 1. Darkness importance (darker = more important)
			darknessImportance := (255.0 - grayVal) / 255.0

			// 2. Edge importance
			edgeImportance := edgeVal / 255.0

			// 3. Center weighting - concentrate lines in center (like v6)
			ddx := float64(x) - centerX
			ddy := float64(y) - centerY
			dist := math.Sqrt(ddx*ddx+ddy*ddy) / maxDist
			centerWeight := math.Exp(-2.0 * dist * dist)

			// Combine: darkness primary, edges secondary (like v6)
			base := darknessImportance*0.6 + edgeImportance*0.4
			importance[y][x] = base * (0.5 + 0.5*centerWeight)

			// Minimum importance (like v6)
			if importance[y][x] < 0.05 {
				importance[y][x] = 0.05
			}
		}
	}

	return importance
}

// findBestLineV8 finds the best next line using enhanced perceptual scoring
func findBestLineV8(fromPin int, hiPins []Pin, hiCanvas [][]uint8, dsCanvas, target [][]float64,
	edgeDisplay, importance [][]float64, config *Config, usedLines map[[2]int]int,
	ssf, displayRes, hiRes int) Line {

	numPins := len(hiPins)

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

		if usedLines[key] >= 4 {
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

				score := evaluateLineV8(hiPins[fromPin], hiPins[task.toPin],
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

// evaluateLineV8 evaluates a line using enhanced perceptual scoring
func evaluateLineV8(from, to Pin, hiCanvas [][]uint8, dsCanvas, target [][]float64,
	edgeDisplay, importance [][]float64, edgeWeight float64, ssf, displayRes, hiRes int) float64 {

	x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
	y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
	x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
	y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

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
			// Mild over-darkening penalty (1.2x like v6)
			// With supersampled rendering, over-darkening is less visible
			totalScore += improvement * imp * 1.2
			worseningPixels++
		}
		totalWeight += imp
	}

	if totalWeight <= 0 {
		return 0
	}

	score := totalScore / totalWeight

	// Worsen ratio penalty (like v6)
	totalPixels := improvingPixels + worseningPixels
	if totalPixels > 0 {
		worsenRatio := float64(worseningPixels) / float64(totalPixels)
		if worsenRatio > 0.4 {
			score *= (1.0 - worsenRatio)
		}
	}

	return score
}

// updateDSCanvasForLineV8 efficiently updates downsampled canvas for a single line
func updateDSCanvasForLineV8(dsCanvas [][]float64, hiCanvas [][]uint8, from, to Pin, ssf, displayRes, hiRes int) {
	x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
	y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
	x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
	y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

	affected := make(map[[2]int]int)

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
			key := [2]int{dpx, dpy}
			affected[key]++
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

	ssfSq := float64(ssf * ssf)
	for key, count := range affected {
		px, py := key[0], key[1]
		delta := 255.0 * float64(count) / ssfSq
		dsCanvas[py][px] -= delta
		if dsCanvas[py][px] < 0 {
			dsCanvas[py][px] = 0
		}
	}
}

// addRemoveOptimizationV8 implements add/remove optimization
func addRemoveOptimizationV8(lines []Line, hiPins []Pin, hiCanvas [][]uint8,
	target, importance, edgeDisplay [][]float64, config *Config,
	ssf, displayRes, hiRes int) []Line {

	if len(lines) == 0 {
		return lines
	}

	maxRemovals := len(lines) / 6 // Allow removing up to 1/6 of lines

	dsCanvas := downsampleCanvas(hiCanvas, ssf, displayRes)
	currentSSIM := quickSSIM(dsCanvas, target, displayRes)
	currentMSE := calculateMSEV6(dsCanvas, target, displayRes)
	fmt.Printf("Before add/remove: MSE = %.2f, SSIM~%.3f\n", currentMSE, currentSSIM)

	type removalCandidate struct {
		index       int
		improvement float64
	}

	candidates := make([]removalCandidate, 0)
	ssfSq := float64(ssf * ssf)

	// Evaluate each line for potential removal
	for idx, line := range lines {
		if line.From < 0 || line.To < 0 {
			continue
		}

		from := hiPins[line.From]
		to := hiPins[line.To]
		x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
		y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
		x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
		y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

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

		// Calculate error improvement from removal (lightening)
		errorChange := 0.0
		for key, info := range blocks {
			px, py := key[0], key[1]
			currentVal := dsCanvas[py][px]
			targetVal := target[py][px]

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

	// Remove lines that improve quality
	for _, c := range candidates {
		if removedCount >= maxRemovals {
			break
		}

		line := lines[c.index]
		from := hiPins[line.From]
		to := hiPins[line.To]

		x0 := clampInt(int(math.Round(from.X)), 0, hiRes-1)
		y0 := clampInt(int(math.Round(from.Y)), 0, hiRes-1)
		x1 := clampInt(int(math.Round(to.X)), 0, hiRes-1)
		y1 := clampInt(int(math.Round(to.Y)), 0, hiRes-1)

		// Remove line from hi-res canvas
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
	newSSIM := quickSSIM(dsCanvas, target, displayRes)
	fmt.Printf("After add/remove: MSE = %.2f, SSIM~%.3f (removed %d lines)\n", newMSE, newSSIM, removedCount)

	return newLines
}