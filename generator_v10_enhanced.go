package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV10Enhanced implements an enhanced version of V10 with:
// 1. Multi-start: try multiple starting pins, keep best result
// 2. Iterative line replacement: after greedy, try replacing each line with a better one
// 3. Adaptive over-darkening penalty: stronger penalty as canvas gets darker
// 4. Higher line reuse in dark areas (up to 6 for very dark regions)
// 5. Better stagnation detection with restart from best pin
func GenerateStringArtV10Enhanced(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v10.1 Enhanced ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)

	alpha := 0.25
	fmt.Printf("Source-over alpha: %.3f\n", alpha)

	// Create enhanced importance map with local contrast awareness
	importance := createImportanceMapV10Enhanced(img, edgeMap, width, height)

	// Pre-compute line pixels
	fmt.Println("Pre-computing line pixels (Bresenham, 1px)...")
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := centerX - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	linePixels := precomputeLinePixelsBresenham(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	// Prepare target
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y) / 255.0
		}
	}

	// Find best starting pin (darkest region)
	bestStartPins := findBestStartPins(target, pins, width, height, 5)
	fmt.Printf("Best starting pins: %v\n", bestStartPins)

	// Try multiple starting pins, keep best result
	var bestLines []Line
	var bestCanvas [][]float64
	bestMSE := math.MaxFloat64

	for attempt, startPin := range bestStartPins {
		fmt.Printf("\n--- Attempt %d: Starting from pin %d ---\n", attempt+1, startPin)

		// Initialize canvas
		canvas := make([][]float64, height)
		for y := 0; y < height; y++ {
			canvas[y] = make([]float64, width)
			for x := 0; x < width; x++ {
				canvas[y][x] = 1.0
			}
		}

		lines := greedyPhaseEnhanced(startPin, pins, canvas, target, importance,
			linePixels, config, width, height, alpha)

		// Line removal
		lines = lineRemovalV10(lines, canvas, target, importance, linePixels, config, width, height, alpha)

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
	bestLines, bestCanvas = iterativeReplacement(bestLines, bestCanvas, target, importance,
		linePixels, pins, config, width, height, alpha)

	// Convert canvas to [0,255]
	fmt.Println("Converting canvas to output format...")
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

// findBestStartPins finds pins near the darkest regions of the image
func findBestStartPins(target [][]float64, pins []Pin, width, height, count int) []int {
	type pinScore struct {
		pin   int
		score float64
	}

	scores := make([]pinScore, len(pins))
	for i, p := range pins {
		// Sample darkness in a radius around the pin
		px, py := int(p.X), int(p.Y)
		totalDark := 0.0
		samples := 0
		for dy := -30; dy <= 30; dy += 5 {
			for dx := -30; dx <= 30; dx += 5 {
				x, y := px+dx, py+dy
				if x >= 0 && x < width && y >= 0 && y < height {
					totalDark += (1.0 - target[y][x])
					samples++
				}
			}
		}
		if samples > 0 {
			scores[i] = pinScore{pin: i, score: totalDark / float64(samples)}
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	result := make([]int, count)
	// Pick diverse pins (not too close to each other)
	result[0] = scores[0].pin
	picked := 1
	for _, s := range scores[1:] {
		if picked >= count {
			break
		}
		// Ensure at least 30 pins apart
		tooClose := false
		for _, r := range result[:picked] {
			dist := s.pin - r
			if dist < 0 {
				dist = -dist
			}
			if dist < 30 {
				tooClose = true
				break
			}
		}
		if !tooClose {
			result[picked] = s.pin
			picked++
		}
	}

	return result[:picked]
}

// greedyPhaseEnhanced is the greedy line selection with improvements
func greedyPhaseEnhanced(startPin int, pins []Pin, canvas, target, importance [][]float64,
	linePixels map[[2]int][][2]int, config *Config, width, height int, alpha float64) []Line {

	lines := make([]Line, 0, config.NumLines)
	currentPin := startPin
	usedLines := make(map[[2]int]int)

	fmt.Println("Phase 1: Enhanced Greedy Add...")
	recentScores := make([]float64, 0, 100)
	stagnationCount := 0
	bestPinAfterStagnation := currentPin

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV10Enhanced(currentPin, pins, canvas, target, importance,
			linePixels, config, usedLines, width, height, alpha)

		if bestLine.Score <= 0.0001 {
			// Try jumping to a different pin before giving up
			if stagnationCount < 5 {
				stagnationCount++
				// Find pin with highest remaining error
				bestPinAfterStagnation = findHighErrorPin(canvas, target, importance, pins, width, height)
				currentPin = bestPinAfterStagnation
				continue
			}
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		stagnationCount = 0

		// Stagnation detection
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
				// Jump to high-error pin instead of stopping
				bestPinAfterStagnation = findHighErrorPin(canvas, target, importance, pins, width, height)
				if bestPinAfterStagnation != currentPin {
					currentPin = bestPinAfterStagnation
					recentScores = recentScores[:0]
					continue
				}
				fmt.Printf("Stopping at line %d (quality plateau)\n", i)
				break
			}
		}

		// Apply line
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		pixels := linePixels[key]
		for _, px := range pixels {
			if px[0] >= 0 && px[0] < width && px[1] >= 0 && px[1] < height {
				canvas[px[1]][px[0]] *= (1.0 - alpha)
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

// findHighErrorPin finds the pin closest to the area with highest remaining error
func findHighErrorPin(canvas, target, importance [][]float64, pins []Pin, width, height int) int {
	// Find the region with highest weighted error
	blockSize := 40
	bestError := 0.0
	bestX, bestY := width/2, height/2

	for by := 0; by < height; by += blockSize {
		for bx := 0; bx < width; bx += blockSize {
			blockError := 0.0
			count := 0
			for y := by; y < by+blockSize && y < height; y++ {
				for x := bx; x < bx+blockSize && x < width; x++ {
					diff := canvas[y][x] - target[y][x]
					if diff > 0 { // Only care about too-bright areas (need more lines)
						blockError += diff * diff * importance[y][x]
						count++
					}
				}
			}
			if count > 0 && blockError/float64(count) > bestError {
				bestError = blockError / float64(count)
				bestX = bx + blockSize/2
				bestY = by + blockSize/2
			}
		}
	}

	// Find closest pin to this region
	bestPin := 0
	bestDist := math.MaxFloat64
	for i, p := range pins {
		dx := p.X - float64(bestX)
		dy := p.Y - float64(bestY)
		dist := dx*dx + dy*dy
		if dist < bestDist {
			bestDist = dist
			bestPin = i
		}
	}

	return bestPin
}

// findBestLineV10Enhanced finds the best line with adaptive penalties
func findBestLineV10Enhanced(fromPin int, pins []Pin, canvas, target [][]float64,
	importance [][]float64, linePixels map[[2]int][][2]int, config *Config, usedLines map[[2]int]int,
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

		// Adaptive reuse limit: allow more reuse for lines in dark areas
		maxReuse := 4
		pixels := linePixels[key]
		if len(pixels) > 0 {
			avgTarget := 0.0
			for _, px := range pixels {
				if px[0] >= 0 && px[0] < width && px[1] >= 0 && px[1] < height {
					avgTarget += target[px[1]][px[0]]
				}
			}
			avgTarget /= float64(len(pixels))
			if avgTarget < 0.3 { // Very dark area
				maxReuse = 6
			} else if avgTarget < 0.5 { // Dark area
				maxReuse = 5
			}
		}

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

				score := evaluateLineV10Enhanced(task.key, canvas, target, importance,
					linePixels, width, height, alpha)

				// Progressive reuse penalty
				if usedLines[task.key] > 0 {
					penalty := 1.0 / (1.0 + 0.7*float64(usedLines[task.key]))
					score *= penalty
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

// evaluateLineV10Enhanced evaluates with adaptive over-darkening penalty
func evaluateLineV10Enhanced(key [2]int, canvas, target [][]float64,
	importance [][]float64, linePixels map[[2]int][][2]int,
	width, height int, alpha float64) float64 {

	pixels := linePixels[key]
	if len(pixels) == 0 {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0

	for _, px := range pixels {
		x, y := px[0], px[1]
		if x < 0 || x >= width || y < 0 || y >= height {
			continue
		}

		currentVal := canvas[y][x]
		targetVal := target[y][x]

		// Source-over: new = old * (1 - alpha)
		newVal := currentVal * (1.0 - alpha)

		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)
		improvement := oldError - newError

		imp := importance[y][x]

		if improvement > 0 {
			totalScore += improvement * imp
			improvingPixels++
		} else if improvement < 0 {
			// Adaptive over-darkening penalty:
			// Stronger penalty when canvas is already darker than target
			overDarkAmount := targetVal - newVal
			if overDarkAmount > 0.3 {
				// Very over-dark: heavy penalty
				totalScore += improvement * imp * 2.0
			} else if overDarkAmount > 0.15 {
				// Moderately over-dark
				totalScore += improvement * imp * 1.5
			} else {
				// Slightly over-dark: mild penalty (allow some for coverage)
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

	// Penalize lines where too many pixels get worse
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

// iterativeReplacement tries to replace each line with a better alternative
func iterativeReplacement(lines []Line, canvas, target, importance [][]float64,
	linePixels map[[2]int][][2]int, pins []Pin, config *Config,
	width, height int, alpha float64) ([]Line, [][]float64) {

	beforeMSE := calculateMSENorm(canvas, target, width, height) * 255 * 255
	fmt.Printf("Before replacement: MSE = %.2f (%d lines)\n", beforeMSE, len(lines))

	replacements := 0
	maxIterations := 2 // Do 2 passes of replacement

	for iter := 0; iter < maxIterations; iter++ {
		iterReplacements := 0

		for idx := range lines {
			line := lines[idx]
			if line.From < 0 || line.To < 0 {
				continue
			}

			key := [2]int{min(line.From, line.To), max(line.From, line.To)}
			pixels := linePixels[key]

			// Remove this line temporarily
			for _, px := range pixels {
				x, y := px[0], px[1]
				if x >= 0 && x < width && y >= 0 && y < height {
					canvas[y][x] /= (1.0 - alpha)
					if canvas[y][x] > 1.0 {
						canvas[y][x] = 1.0
					}
				}
			}

			// Find best replacement from the same starting pin
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

				score := evaluateLineV10Enhanced(newKey, canvas, target, importance,
					linePixels, width, height, alpha)

				if score > bestScore {
					bestScore = score
					bestTo = toPin
				}
			}

			// Also evaluate the original line
			origScore := evaluateLineV10Enhanced(key, canvas, target, importance,
				linePixels, width, height, alpha)

			if bestTo >= 0 && bestScore > origScore*1.1 { // Must be 10% better to replace
				// Replace with better line
				newKey := [2]int{min(line.From, bestTo), max(line.From, bestTo)}
				newPixels := linePixels[newKey]
				for _, px := range newPixels {
					x, y := px[0], px[1]
					if x >= 0 && x < width && y >= 0 && y < height {
						canvas[y][x] *= (1.0 - alpha)
					}
				}
				lines[idx] = Line{From: line.From, To: bestTo, Score: bestScore}
				iterReplacements++
			} else {
				// Keep original line
				for _, px := range pixels {
					x, y := px[0], px[1]
					if x >= 0 && x < width && y >= 0 && y < height {
						canvas[y][x] *= (1.0 - alpha)
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

// createImportanceMapV10Enhanced creates an enhanced importance map
// with local contrast awareness and structure preservation
func createImportanceMapV10Enhanced(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	centerX, centerY := float64(width)/2, float64(height)/2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	// Compute local contrast (variance in 5x5 neighborhood)
	localContrast := make([][]float64, height)
	for y := 0; y < height; y++ {
		localContrast[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			sum := 0.0
			sumSq := 0.0
			count := 0
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < height && nx >= 0 && nx < width {
						v := float64(img.GrayAt(nx, ny).Y) / 255.0
						sum += v
						sumSq += v * v
						count++
					}
				}
			}
			if count > 0 {
				mean := sum / float64(count)
				variance := sumSq/float64(count) - mean*mean
				if variance < 0 {
					variance = 0
				}
				localContrast[y][x] = math.Sqrt(variance)
			}
		}
	}

	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grayVal := float64(img.GrayAt(x, y).Y)
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)

			// Darkness importance
			darknessImp := (255.0 - grayVal) / 255.0

			// Edge importance
			edgeImp := edgeVal / 255.0

			// Local contrast importance (high contrast areas are perceptually important)
			contrastImp := localContrast[y][x] * 3.0 // Scale up
			if contrastImp > 1.0 {
				contrastImp = 1.0
			}

			// Center weighting
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist
			centerWeight := 0.4 + 0.6*math.Exp(-1.0*dist*dist)

			// Combine: darkness primary, edges + contrast secondary
			base := darknessImp*0.5 + edgeImp*0.3 + contrastImp*0.2
			importance[y][x] = base * centerWeight

			if importance[y][x] < 0.01 {
				importance[y][x] = 0.01
			}
		}
	}

	return importance
}
