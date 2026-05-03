package main

import (
	"fmt"
	"image"
	"math"
	"sort"
	"sync"
)

// GenerateStringArtV5 is the improved string art generator
// Key improvements over previous versions:
// 1. Uses RAW grayscale as target (not edge-blended)
// 2. Proper error reduction scoring (measures actual improvement)
// 3. Anti-aliased line rendering (Xiaolin Wu's algorithm)
// 4. Fear removal: ignores pixels that temporarily worsen
// 5. Importance map with center weighting
// 6. Line removal pass for quality refinement
// 7. Adaptive line weight based on progress
func GenerateStringArtV5(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Create target array (what we want to achieve)
	// 0 = black (dark), 255 = white (light)
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

	// Create importance map
	importance := createImportanceMapV5(img, edgeMap, width, height)

	// Pre-compute all line pixels with anti-aliasing
	fmt.Println("Pre-computing line pixels with anti-aliasing...")
	linePixels := precomputeLinePixels(pins, width, height, config.MinDistance)

	fmt.Printf("=== String Art Generator v5.0 ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int) // Track how many times each line is used

	// Adaptive parameters
	baseWeight := float64(config.LineWeight)
	recentScores := make([]float64, 0, 20)
	stagnationCount := 0

	for i := 0; i < config.NumLines; i++ {
		// Adaptive line weight: reduce as we progress to capture fine details
		// With source-over compositing, weight/255 = per-line alpha
		// weight 30 → α ≈ 0.118 (each line removes ~12% of remaining brightness)
		// weight 15 → α ≈ 0.059 (each line removes ~6% - finer control)
		progress := float64(i) / float64(config.NumLines)
		adaptiveWeight := baseWeight * (1.0 - 0.5*progress) // Start at 100%, end at 50%
		if adaptiveWeight < 10 {
			adaptiveWeight = 10
		}

		bestLine := findBestLineV5(currentPin, pins, canvas, target, edgeMap, importance,
			linePixels, config, adaptiveWeight, usedLines)

		if bestLine.Score <= 0.1 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Adaptive stopping: check for stagnation
		recentScores = append(recentScores, bestLine.Score)
		if len(recentScores) > 20 {
			recentScores = recentScores[1:]
		}
		if len(recentScores) >= 20 {
			avgScore := 0.0
			for _, s := range recentScores {
				avgScore += s
			}
			avgScore /= float64(len(recentScores))
			if avgScore < config.StopThreshold*0.3 {
				stagnationCount++
				if stagnationCount > 3 {
					fmt.Printf("Stopping at line %d (quality plateau, avg score: %.2f)\n", i, avgScore)
					break
				}
			} else {
				stagnationCount = 0
			}
		}

		// Draw line on canvas with anti-aliasing
		drawLineAAOnCanvas(canvas, pins[bestLine.From], pins[bestLine.To], adaptiveWeight, width, height)

		// Track usage
		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			// Calculate current MSE
			mse := calculateMSE(canvas, target, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, weight: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, adaptiveWeight)
		}
	}

	// Line removal pass: remove lines that hurt quality
	fmt.Println("\n--- Line Removal Pass ---")
	lines, canvas = lineRemovalPass(lines, pins, target, canvas, importance, config, width, height)

	fmt.Printf("\nFinal: %d lines, MSE: %.1f\n", len(lines), calculateMSE(canvas, target, width, height))
	return lines, canvas
}

// AntiAliasedPixel represents a pixel with sub-pixel weight
type AntiAliasedPixel struct {
	X, Y   int
	Weight float64 // 0.0 to 1.0
}

// LineSegment stores pre-computed pixels for a line between two pins
type LineSegment struct {
	From, To int
	Pixels   []AntiAliasedPixel
}

// precomputeLinePixels pre-computes all valid line segments with anti-aliased pixels
func precomputeLinePixels(pins []Pin, width, height, minDistance int) map[[2]int][]AntiAliasedPixel {
	numPins := len(pins)
	result := make(map[[2]int][]AntiAliasedPixel)

	for i := 0; i < numPins; i++ {
		for j := i + 1; j < numPins; j++ {
			// Check distance constraint
			distance := j - i
			if distance < minDistance || numPins-distance < minDistance {
				continue
			}

			pixels := getAntiAliasedLinePixels(pins[i], pins[j], width, height)
			if len(pixels) > 0 {
				result[[2]int{i, j}] = pixels
			}
		}
	}

	return result
}

// getAntiAliasedLinePixels returns pixels along a line using Xiaolin Wu's algorithm
func getAntiAliasedLinePixels(from, to Pin, width, height int) []AntiAliasedPixel {
	var pixels []AntiAliasedPixel

	x0, y0 := from.X, from.Y
	x1, y1 := to.X, to.Y

	steep := math.Abs(y1-y0) > math.Abs(x1-x0)

	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}

	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}

	dx := x1 - x0
	dy := y1 - y0

	var gradient float64
	if dx == 0 {
		gradient = 1.0
	} else {
		gradient = dy / dx
	}

	// First endpoint
	xEnd := math.Round(x0)
	yEnd := y0 + gradient*(xEnd-x0)
	xGap := rfpart(x0 + 0.5)
	xpxl1 := int(xEnd)
	ypxl1 := int(math.Floor(yEnd))

	if steep {
		addAAPixel(&pixels, ypxl1, xpxl1, rfpart(yEnd)*xGap, width, height)
		addAAPixel(&pixels, ypxl1+1, xpxl1, fpart(yEnd)*xGap, width, height)
	} else {
		addAAPixel(&pixels, xpxl1, ypxl1, rfpart(yEnd)*xGap, width, height)
		addAAPixel(&pixels, xpxl1, ypxl1+1, fpart(yEnd)*xGap, width, height)
	}

	intery := yEnd + gradient

	// Second endpoint
	xEnd = math.Round(x1)
	yEnd = y1 + gradient*(xEnd-x1)
	xGap = fpart(x1 + 0.5)
	xpxl2 := int(xEnd)
	ypxl2 := int(math.Floor(yEnd))

	if steep {
		addAAPixel(&pixels, ypxl2, xpxl2, rfpart(yEnd)*xGap, width, height)
		addAAPixel(&pixels, ypxl2+1, xpxl2, fpart(yEnd)*xGap, width, height)
	} else {
		addAAPixel(&pixels, xpxl2, ypxl2, rfpart(yEnd)*xGap, width, height)
		addAAPixel(&pixels, xpxl2, ypxl2+1, fpart(yEnd)*xGap, width, height)
	}

	// Main loop
	if steep {
		for x := xpxl1 + 1; x < xpxl2; x++ {
			iy := int(math.Floor(intery))
			addAAPixel(&pixels, iy, x, rfpart(intery), width, height)
			addAAPixel(&pixels, iy+1, x, fpart(intery), width, height)
			intery += gradient
		}
	} else {
		for x := xpxl1 + 1; x < xpxl2; x++ {
			iy := int(math.Floor(intery))
			addAAPixel(&pixels, x, iy, rfpart(intery), width, height)
			addAAPixel(&pixels, x, iy+1, fpart(intery), width, height)
			intery += gradient
		}
	}

	return pixels
}

func addAAPixel(pixels *[]AntiAliasedPixel, x, y int, weight float64, width, height int) {
	if x >= 0 && x < width && y >= 0 && y < height && weight > 0.01 {
		*pixels = append(*pixels, AntiAliasedPixel{X: x, Y: y, Weight: weight})
	}
}

func fpart(x float64) float64 {
	return x - math.Floor(x)
}

func rfpart(x float64) float64 {
	return 1.0 - fpart(x)
}

// createImportanceMapV5 creates an importance map that prioritizes:
// - Dark areas (subject)
// - Edges (features)
// - Center of image (face likely there)
// - High contrast regions
func createImportanceMapV5(img *image.Gray, edgeMap *image.Gray, width, height int) [][]float64 {
	importance := make([][]float64, height)
	centerX, centerY := float64(width)/2, float64(height)/2
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			grayVal := float64(img.GrayAt(x, y).Y)
			edgeVal := float64(edgeMap.GrayAt(x, y).Y)

			// 1. Darkness importance (darker = more important)
			darknessImportance := (255.0 - grayVal) / 255.0

			// 2. Edge importance
			edgeImportance := edgeVal / 255.0

			// 3. Center weighting (Gaussian falloff)
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist
			centerWeight := math.Exp(-2.0 * dist * dist) // Gaussian, sigma ~0.5

			// 4. Combine: darkness is primary, edges are secondary, center is a multiplier
			base := darknessImportance*0.6 + edgeImportance*0.4
			importance[y][x] = base * (0.5 + 0.5*centerWeight) // Center gets up to 1.0x, edges get 0.5x

			// Minimum importance to avoid completely ignoring areas
			if importance[y][x] < 0.05 {
				importance[y][x] = 0.05
			}
		}
	}

	return importance
}

// findBestLineV5 finds the best next line using error reduction scoring
func findBestLineV5(fromPin int, pins []Pin, canvas, target [][]float64, edgeMap *image.Gray,
	importance [][]float64, linePixels map[[2]int][]AntiAliasedPixel,
	config *Config, lineWeight float64, usedLines map[[2]int]int) Line {

	numPins := len(pins)

	type candidate struct {
		toPin int
		score float64
	}

	// Parallel evaluation
	candidates := make([]candidate, 0, numPins)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Split work across workers
	batchSize := (numPins + config.Workers - 1) / config.Workers

	for w := 0; w < config.Workers; w++ {
		start := w * batchSize
		end := start + batchSize
		if end > numPins {
			end = numPins
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			localCandidates := make([]candidate, 0, end-start)

			for toPin := start; toPin < end; toPin++ {
				if toPin == fromPin {
					continue
				}

				// Get pre-computed pixels
				key := [2]int{min(fromPin, toPin), max(fromPin, toPin)}
				pixels, exists := linePixels[key]
				if !exists {
					continue
				}

				// Penalize overused lines
				usageCount := usedLines[key]
				if usageCount >= 3 {
					continue // Don't use same line more than 3 times
				}

				// Calculate error reduction score
				score := evaluateLineV5(pixels, canvas, target, edgeMap, importance,
					lineWeight, config.EdgeWeight)

				// Apply usage penalty
				if usageCount > 0 {
					score *= 1.0 / (1.0 + 0.5*float64(usageCount))
				}

				if score > 0 {
					localCandidates = append(localCandidates, candidate{toPin: toPin, score: score})
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

	// Find best
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return Line{From: fromPin, To: best.toPin, Score: best.score}
}

// evaluateLineV5 calculates the net error reduction if we draw this line
// Uses SOURCE-OVER compositing model to match SVG rendering.
// Heavily penalizes over-darkening since strokes are opaque.
func evaluateLineV5(pixels []AntiAliasedPixel, canvas, target [][]float64,
	edgeMap *image.Gray, importance [][]float64,
	lineWeight, edgeWeight float64) float64 {

	totalScore := 0.0
	totalWeight := 0.0
	improvingPixels := 0
	worseningPixels := 0

	lineAlpha := lineWeight / 255.0 // Same as in drawLineAAOnCanvas

	for _, p := range pixels {
		x, y := p.X, p.Y
		w := p.Weight

		currentVal := canvas[y][x]  // Current brightness (0=black, 255=white)
		targetVal := target[y][x]   // Target brightness

		// Source-over compositing: new brightness = current * (1 - α*w)
		effectiveAlpha := lineAlpha * w
		if effectiveAlpha > 1.0 {
			effectiveAlpha = 1.0
		}
		newVal := currentVal * (1.0 - effectiveAlpha)

		// Error before and after (squared)
		oldError := (currentVal - targetVal) * (currentVal - targetVal)
		newError := (newVal - targetVal) * (newVal - targetVal)

		// Net improvement (positive = better, negative = worse)
		improvement := oldError - newError

		// Weight by importance
		imp := importance[y][x]

		// Edge bonus for improving pixels only
		edgeVal := float64(edgeMap.GrayAt(x, y).Y) / 255.0
		edgeBonus := 1.0 + edgeVal*edgeWeight*0.2

		if improvement > 0 {
			// This pixel improves
			totalScore += improvement * imp * edgeBonus * w
			improvingPixels++
		} else if improvement < 0 {
			// This pixel gets WORSE (over-darkening)
			// Penalize heavily - with opaque strokes, over-darkening is very visible
			totalScore += improvement * imp * 4.0 * w // 4.0x penalty for worsening
			worseningPixels++
		}

		totalWeight += w
	}

	if totalWeight > 0 {
		score := totalScore / totalWeight

		// Additional penalty if too many pixels worsen
		totalPixels := improvingPixels + worseningPixels
		if totalPixels > 0 {
			worsenRatio := float64(worseningPixels) / float64(totalPixels)
			if worsenRatio > 0.4 {
				// More than 40% of pixels get worse - penalize
				score *= (1.0 - worsenRatio)
			}
		}

		return score
	}
	return 0
}

// drawLineAAOnCanvas draws an anti-aliased line on the canvas
// Uses SOURCE-OVER COMPOSITING to match SVG rendering behavior.
// 
// In SVG, a 0.18mm stroke in viewBox 600 is sub-pixel on most screens.
// The browser anti-aliases it, giving each line a fractional alpha.
// Multiple lines composite via: brightness(N) = (1-α)^N
//
// We simulate this by using multiplicative darkening (not additive).
// This accurately predicts how the SVG will look on screen.
func drawLineAAOnCanvas(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	pixels := getAntiAliasedLinePixels(from, to, width, height)

	// Calculate per-line alpha based on SVG rendering model
	// 0.18mm stroke in 600-unit viewBox = 0.0003 of width
	// On a typical screen (400-800px display), this is 0.12-0.24px
	// We use weight to control the effective alpha
	// Higher weight = thicker effective line = more darkening per pass
	lineAlpha := weight / 255.0 // Normalize: weight 25 → α ≈ 0.098
	
	for _, p := range pixels {
		// Source-over compositing: new_brightness = old_brightness * (1 - α * aa_weight)
		effectiveAlpha := lineAlpha * p.Weight
		if effectiveAlpha > 1.0 {
			effectiveAlpha = 1.0
		}
		canvas[p.Y][p.X] *= (1.0 - effectiveAlpha)
	}
}

// lineRemovalPass removes lines that hurt overall quality
func lineRemovalPass(lines []Line, pins []Pin, target, canvas [][]float64,
	importance [][]float64, config *Config, width, height int) ([]Line, [][]float64) {

	if len(lines) == 0 {
		return lines, canvas
	}

	removedCount := 0
	maxRemovals := len(lines) / 10 // Remove at most 10% of lines

	// Calculate current total error
	currentError := calculateWeightedMSE(canvas, target, importance, width, height)
	fmt.Printf("Before removal: weighted MSE = %.2f\n", currentError)

	// Try removing each line and check if quality improves
	type removalCandidate struct {
		index       int
		improvement float64
	}

	candidates := make([]removalCandidate, 0)

	for idx, line := range lines {
		if line.From < 0 || line.To < 0 {
			continue
		}

		// Simulate removing this line (add back its contribution)
		pixels := getAntiAliasedLinePixels(pins[line.From], pins[line.To], width, height)

		// Calculate how much error would change if we remove this line
		errorChange := 0.0
		for _, p := range pixels {
			currentVal := canvas[p.Y][p.X]
			targetVal := target[p.Y][p.X]

			// If we remove the line, the pixel gets lighter
			restoredVal := currentVal + float64(config.LineWeight)*p.Weight*0.7 // Approximate
			if restoredVal > 255 {
				restoredVal = 255
			}

			oldErr := (currentVal - targetVal) * (currentVal - targetVal)
			newErr := (restoredVal - targetVal) * (restoredVal - targetVal)

			imp := importance[p.Y][p.X]
			errorChange += (oldErr - newErr) * imp
		}

		// If removing the line reduces error (errorChange > 0), it's a candidate
		if errorChange > 0 {
			candidates = append(candidates, removalCandidate{index: idx, improvement: errorChange})
		}
	}

	// Sort by improvement (most beneficial removals first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].improvement > candidates[j].improvement
	})

	// Remove the worst lines
	removedIndices := make(map[int]bool)
	for _, c := range candidates {
		if removedCount >= maxRemovals {
			break
		}

		line := lines[c.index]
		pixels := getAntiAliasedLinePixels(pins[line.From], pins[line.To], width, height)

		// Actually remove (lighten canvas)
		for _, p := range pixels {
			canvas[p.Y][p.X] += float64(config.LineWeight) * p.Weight * 0.7
			if canvas[p.Y][p.X] > 255 {
				canvas[p.Y][p.X] = 255
			}
		}

		removedIndices[c.index] = true
		removedCount++
	}

	// Build new line list without removed lines
	newLines := make([]Line, 0, len(lines)-removedCount)
	for idx, line := range lines {
		if !removedIndices[idx] {
			newLines = append(newLines, line)
		}
	}

	newError := calculateWeightedMSE(canvas, target, importance, width, height)
	fmt.Printf("After removal: weighted MSE = %.2f (removed %d lines)\n", newError, removedCount)

	return newLines, canvas
}

// calculateMSE calculates mean squared error between canvas and target
func calculateMSE(canvas, target [][]float64, width, height int) float64 {
	totalError := 0.0
	count := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			totalError += diff * diff
			count++
		}
	}

	if count > 0 {
		return totalError / float64(count)
	}
	return 0
}

// calculateWeightedMSE calculates importance-weighted MSE
func calculateWeightedMSE(canvas, target, importance [][]float64, width, height int) float64 {
	totalError := 0.0
	totalWeight := 0.0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			w := importance[y][x]
			totalError += diff * diff * w
			totalWeight += w
		}
	}

	if totalWeight > 0 {
		return totalError / totalWeight
	}
	return 0
}

// ExportSVGV5 exports with float64 canvas
func ExportSVGV5(lines []Line, config *Config, outputPath string) error {
	return ExportSVG(lines, config, outputPath)
}

// RenderCanvasV5ToImage converts float64 canvas to PNG
func RenderCanvasV5ToImage(canvas [][]float64, outputPath string) error {
	height := len(canvas)
	width := len(canvas[0])

	intCanvas := make([][]int, height)
	for y := 0; y < height; y++ {
		intCanvas[y] = make([]int, width)
		for x := 0; x < width; x++ {
			val := int(math.Round(canvas[y][x]))
			if val < 0 {
				val = 0
			}
			if val > 255 {
				val = 255
			}
			intCanvas[y][x] = val
		}
	}

	return RenderCanvasToImage(intCanvas, outputPath)
}
