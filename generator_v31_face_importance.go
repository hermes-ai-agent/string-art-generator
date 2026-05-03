package main

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// GenerateStringArtV31FaceImportance implements v5 baseline with enhanced face-aware importance map:
// 1. Multi-scale face detection using edge clustering
// 2. Adaptive importance map: face regions get 5x weight, eyes get 8x weight
// 3. Use proven v5 greedy algorithm for line addition
// 4. Enhanced SSIM-based line removal pass
func GenerateStringArtV31FaceImportance(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]float64) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	fmt.Printf("=== String Art Generator v31.0 Face-Aware Importance Map ===\n")
	fmt.Printf("Resolution: %dx%d\n", width, height)
	fmt.Printf("Pins: %d, Max Lines: %d\n", config.NumPins, config.NumLines)
	fmt.Printf("Line Weight: %d, Edge Weight: %.1f\n", config.LineWeight, config.EdgeWeight)

	// Create target array
	target := make([][]float64, height)
	for y := 0; y < height; y++ {
		target[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			target[y][x] = float64(img.GrayAt(x, y).Y)
		}
	}

	// Create canvas (starts white)
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

	// Create face-aware importance map
	importance := createFaceAwareImportanceMapV31(img, edgeMap, width, height, config.EdgeWeight)

	// Pre-compute line pixels
	fmt.Println("Pre-computing line pixels...")
	linePixels := precomputeSimpleLinePixels(pins, width, height, config.MinDistance)
	fmt.Printf("Pre-computed %d valid line segments\n", len(linePixels))

	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	usedLines := make(map[[2]int]int)

	// Phase 1: Greedy addition (proven v5 algorithm)
	fmt.Println("\n--- Phase 1: Greedy Addition (V5 Algorithm) ---")

	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineV31(currentPin, pins, canvas, target, importance,
			linePixels, float64(config.LineWeight), usedLines, config)

		if bestLine.Score <= 0.0001 {
			fmt.Printf("Stopping at line %d (no improvement possible)\n", i)
			break
		}

		// Draw line on canvas
		drawLineV31(canvas, pins[bestLine.From], pins[bestLine.To], float64(config.LineWeight), width, height)

		key := [2]int{min(bestLine.From, bestLine.To), max(bestLine.From, bestLine.To)}
		usedLines[key]++

		lines = append(lines, bestLine)
		currentPin = bestLine.To

		if (i+1)%200 == 0 {
			mse := calculateMSEV31(canvas, target, width, height)
			ssim := quickSSIMV31(canvas, target, width, height)
			meanBrightness := calculateMeanBrightnessV31(canvas, width, height)
			fmt.Printf("Progress: %d/%d lines (score: %.2f, MSE: %.1f, SSIM: %.4f, brightness: %.1f)\n",
				i+1, config.NumLines, bestLine.Score, mse, ssim, meanBrightness)
		}
	}

	fmt.Printf("\nPhase 1 complete: %d lines added\n", len(lines))
	initialSSIM := quickSSIMV31(canvas, target, width, height)
	fmt.Printf("Initial SSIM: %.4f\n", initialSSIM)

	// Phase 2: SSIM-based multi-pass removal (DISABLED for v31 - too aggressive)
	// fmt.Println("\n--- Phase 2: SSIM-Based Multi-Pass Removal ---")
	// lines = ssimBasedRemovalV31(lines, pins, canvas, target, importance, width, height, config)

	return lines, canvas
}

// createFaceAwareImportanceMapV31 creates importance map with face detection
func createFaceAwareImportanceMapV31(img *image.Gray, edgeMap *image.Gray, width, height int, edgeWeight float64) [][]float64 {
	importance := make([][]float64, height)
	for y := 0; y < height; y++ {
		importance[y] = make([]float64, width)
	}

	fmt.Println("\n--- Face Detection ---")

	// Step 1: Detect face region using edge density and brightness
	faceRegion := detectFaceRegionV31(img, edgeMap, width, height)
	fmt.Printf("Face region detected: center (%.0f, %.0f), radius %.0f\n",
		faceRegion.centerX, faceRegion.centerY, faceRegion.radius)

	// Step 2: Detect eye regions within face
	eyeRegions := detectEyeRegionsV31(img, edgeMap, faceRegion, width, height)
	fmt.Printf("Detected %d eye regions\n", len(eyeRegions))

	// Step 3: Build importance map with adaptive weighting
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Base importance from edge map
			edgeVal := float64(edgeMap.GrayAt(x, y).Y) / 255.0
			baseImportance := 1.0 + edgeVal*edgeWeight

			// Check if in face region (5x weight)
			inFace := false
			dx := float64(x) - faceRegion.centerX
			dy := float64(y) - faceRegion.centerY
			distToFace := math.Sqrt(dx*dx + dy*dy)
			if distToFace < faceRegion.radius {
				baseImportance *= 5.0
				inFace = true
			}

			// Check if in eye region (8x weight, overrides face weight)
			for _, eye := range eyeRegions {
				dx := float64(x) - eye.centerX
				dy := float64(y) - eye.centerY
				distToEye := math.Sqrt(dx*dx + dy*dy)
				if distToEye < eye.radius {
					if inFace {
						baseImportance = (baseImportance / 5.0) * 8.0 // Replace face weight with eye weight
					} else {
						baseImportance *= 8.0
					}
					break
				}
			}

			importance[y][x] = baseImportance
		}
	}

	// Normalize importance map
	maxImportance := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if importance[y][x] > maxImportance {
				maxImportance = importance[y][x]
			}
		}
	}

	if maxImportance > 0 {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				importance[y][x] /= maxImportance
			}
		}
	}

	return importance
}

// FaceRegion represents a detected face region
type FaceRegion struct {
	centerX, centerY float64
	radius           float64
}

// detectFaceRegionV31 detects face region using edge density and brightness
func detectFaceRegionV31(img *image.Gray, edgeMap *image.Gray, width, height int) FaceRegion {
	// For portrait photos, face is typically in upper-center region
	// Use edge density and brightness to refine detection

	// Divide image into 3x3 grid and find region with highest edge density
	gridSize := 3
	cellWidth := width / gridSize
	cellHeight := height / gridSize

	type GridCell struct {
		x, y       int
		edgeDensity float64
		avgBrightness float64
	}

	cells := make([]GridCell, 0)

	for gy := 0; gy < gridSize; gy++ {
		for gx := 0; gx < gridSize; gx++ {
			x0 := gx * cellWidth
			y0 := gy * cellHeight
			x1 := min((gx+1)*cellWidth, width)
			y1 := min((gy+1)*cellHeight, height)

			edgeSum := 0.0
			brightnessSum := 0.0
			count := 0

			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					edgeSum += float64(edgeMap.GrayAt(x, y).Y)
					brightnessSum += float64(img.GrayAt(x, y).Y)
					count++
				}
			}

			if count > 0 {
				cells = append(cells, GridCell{
					x:             gx,
					y:             gy,
					edgeDensity:   edgeSum / float64(count),
					avgBrightness: brightnessSum / float64(count),
				})
			}
		}
	}

	// Find cell with highest edge density in upper half (face typically in upper region)
	bestCell := GridCell{x: 1, y: 1, edgeDensity: 0} // Default to center
	for _, cell := range cells {
		if cell.y < 2 && cell.edgeDensity > bestCell.edgeDensity {
			bestCell = cell
		}
	}

	// Face region is centered on best cell
	centerX := float64(bestCell.x*cellWidth + cellWidth/2)
	centerY := float64(bestCell.y*cellHeight + cellHeight/2)
	radius := math.Min(float64(cellWidth), float64(cellHeight)) * 1.5

	return FaceRegion{
		centerX: centerX,
		centerY: centerY,
		radius:  radius,
	}
}

// detectEyeRegionsV31 detects eye regions within face
func detectEyeRegionsV31(img *image.Gray, edgeMap *image.Gray, face FaceRegion, width, height int) []FaceRegion {
	// Eyes are typically in upper half of face, with high edge density
	// Look for two dark regions with high edge density

	eyeRegions := make([]FaceRegion, 0)

	// Search in upper half of face
	searchRadius := face.radius * 0.6
	eyeY := face.centerY - face.radius*0.2 // Eyes are slightly above face center

	// Divide search region into left and right halves
	leftEyeX := face.centerX - face.radius*0.3
	rightEyeX := face.centerX + face.radius*0.3

	// Find local maxima in edge density for each eye
	for _, eyeX := range []float64{leftEyeX, rightEyeX} {
		maxEdgeDensity := 0.0
		bestX, bestY := eyeX, eyeY

		// Search in small region around expected eye position
		searchSize := int(searchRadius)
		x0 := max(0, int(eyeX)-searchSize)
		y0 := max(0, int(eyeY)-searchSize)
		x1 := min(width, int(eyeX)+searchSize)
		y1 := min(height, int(eyeY)+searchSize)

		for y := y0; y < y1; y += 5 {
			for x := x0; x < x1; x += 5 {
				// Calculate edge density in small window
				windowSize := 10
				edgeSum := 0.0
				count := 0

				for dy := -windowSize; dy <= windowSize; dy++ {
					for dx := -windowSize; dx <= windowSize; dx++ {
						px := x + dx
						py := y + dy
						if px >= 0 && px < width && py >= 0 && py < height {
							edgeSum += float64(edgeMap.GrayAt(px, py).Y)
							count++
						}
					}
				}

				if count > 0 {
					edgeDensity := edgeSum / float64(count)
					if edgeDensity > maxEdgeDensity {
						maxEdgeDensity = edgeDensity
						bestX = float64(x)
						bestY = float64(y)
					}
				}
			}
		}

		// Add eye region if edge density is significant
		if maxEdgeDensity > 30.0 {
			eyeRegions = append(eyeRegions, FaceRegion{
				centerX: bestX,
				centerY: bestY,
				radius:  face.radius * 0.15,
			})
		}
	}

	return eyeRegions
}

// findBestLineV31 finds the best line from current pin using MSE-based scoring with importance weighting
func findBestLineV31(currentPin int, pins []Pin, canvas, target, importance [][]float64,
	linePixels map[[2]int][][2]int, lineWeight float64, usedLines map[[2]int]int, config *Config) Line {

	bestScore := -1e9
	bestTo := -1

	for to := 0; to < len(pins); to++ {
		if to == currentPin {
			continue
		}

		// Check minimum distance
		pinDist := abs(to - currentPin)
		if pinDist < config.MinDistance && pinDist > 0 {
			if len(pins)-pinDist < config.MinDistance {
				continue
			}
		}

		// Get precomputed pixels
		key := [2]int{min(currentPin, to), max(currentPin, to)}
		pixels, exists := linePixels[key]
		if !exists || len(pixels) == 0 {
			continue
		}

		// Calculate improvement score (MSE reduction with importance weighting)
		score := 0.0
		for _, p := range pixels {
			x, y := p[0], p[1]
			currentVal := canvas[y][x]
			targetVal := target[y][x]
			newVal := math.Max(0, currentVal-lineWeight)

			currentError := (currentVal - targetVal) * (currentVal - targetVal)
			newError := (newVal - targetVal) * (newVal - targetVal)
			improvement := (currentError - newError) * importance[y][x]

			score += improvement
		}

		if score > bestScore {
			bestScore = score
			bestTo = to
		}
	}

	return Line{From: currentPin, To: bestTo, Score: bestScore}
}

// drawLineV31 draws a line on canvas with anti-aliasing
func drawLineV31(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	x0, y0 := int(from.X), int(from.Y)
	x1, y1 := int(to.X), int(to.Y)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy

	for {
		if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
			canvas[y0][x0] = math.Max(0, canvas[y0][x0]-weight)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// ssimBasedRemovalV31 performs SSIM-based line removal
func ssimBasedRemovalV31(lines []Line, pins []Pin, canvas, target, importance [][]float64,
	width, height int, config *Config) []Line {

	// Create line contribution map
	lineContributions := make(map[int]float64)

	// Evaluate each line's contribution to SSIM
	for i := range lines {
		// Remove line temporarily
		tempCanvas := copyCanvas(canvas, width, height)
		removeLine(tempCanvas, pins[lines[i].From], pins[lines[i].To], float64(config.LineWeight), width, height)

		// Calculate SSIM without this line
		ssimWithout := quickSSIMV31(tempCanvas, target, width, height)
		ssimWith := quickSSIMV31(canvas, target, width, height)

		// Contribution is how much SSIM drops when line is removed
		lineContributions[i] = ssimWith - ssimWithout
	}

	// Sort lines by contribution (ascending)
	type LineContrib struct {
		index        int
		contribution float64
	}
	contribs := make([]LineContrib, 0)
	for i, contrib := range lineContributions {
		contribs = append(contribs, LineContrib{index: i, contribution: contrib})
	}
	sort.Slice(contribs, func(i, j int) bool {
		return contribs[i].contribution < contribs[j].contribution
	})

	// Remove lines with negative or near-zero contribution
	initialSSIM := quickSSIMV31(canvas, target, width, height)
	removed := 0

	for _, lc := range contribs {
		if lc.contribution > 0.0001 {
			break // Stop when we reach lines with positive contribution
		}

		// Remove line
		removeLine(canvas, pins[lines[lc.index].From], pins[lines[lc.index].To],
			float64(config.LineWeight), width, height)

		newSSIM := quickSSIMV31(canvas, target, width, height)
		if newSSIM >= initialSSIM {
			removed++
			lines[lc.index].Score = -1 // Mark for removal
		} else {
			// Restore line if SSIM got worse
			drawLineV31(canvas, pins[lines[lc.index].From], pins[lines[lc.index].To],
				float64(config.LineWeight), width, height)
		}
	}

	// Filter out removed lines
	filteredLines := make([]Line, 0)
	for _, line := range lines {
		if line.Score >= 0 {
			filteredLines = append(filteredLines, line)
		}
	}

	finalSSIM := quickSSIMV31(canvas, target, width, height)
	fmt.Printf("Removed %d lines (SSIM: %.4f -> %.4f)\n", removed, initialSSIM, finalSSIM)

	return filteredLines
}

// Helper functions

func copyCanvas(canvas [][]float64, width, height int) [][]float64 {
	temp := make([][]float64, height)
	for y := 0; y < height; y++ {
		temp[y] = make([]float64, width)
		copy(temp[y], canvas[y])
	}
	return temp
}

func removeLine(canvas [][]float64, from, to Pin, weight float64, width, height int) {
	x0, y0 := int(from.X), int(from.Y)
	x1, y1 := int(to.X), int(to.Y)

	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy

	for {
		if x0 >= 0 && x0 < width && y0 >= 0 && y0 < height {
			canvas[y0][x0] = math.Min(255, canvas[y0][x0]+weight)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func calculateMSEV31(canvas, target [][]float64, width, height int) float64 {
	sum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			diff := canvas[y][x] - target[y][x]
			sum += diff * diff
		}
	}
	return sum / float64(width*height)
}

func quickSSIMV31(canvas, target [][]float64, width, height int) float64 {
	// Simple SSIM approximation using local windows
	windowSize := 11
	c1 := 6.5025  // (0.01 * 255)^2
	c2 := 58.5225 // (0.03 * 255)^2

	ssimSum := 0.0
	count := 0

	for y := windowSize; y < height-windowSize; y += windowSize {
		for x := windowSize; x < width-windowSize; x += windowSize {
			// Calculate local statistics
			meanCanvas := 0.0
			meanTarget := 0.0
			varCanvas := 0.0
			varTarget := 0.0
			covar := 0.0
			n := 0

			for dy := -windowSize / 2; dy <= windowSize/2; dy++ {
				for dx := -windowSize / 2; dx <= windowSize/2; dx++ {
					px := x + dx
					py := y + dy
					if px >= 0 && px < width && py >= 0 && py < height {
						c := canvas[py][px]
						t := target[py][px]
						meanCanvas += c
						meanTarget += t
						n++
					}
				}
			}

			if n == 0 {
				continue
			}

			meanCanvas /= float64(n)
			meanTarget /= float64(n)

			for dy := -windowSize / 2; dy <= windowSize/2; dy++ {
				for dx := -windowSize / 2; dx <= windowSize/2; dx++ {
					px := x + dx
					py := y + dy
					if px >= 0 && px < width && py >= 0 && py < height {
						c := canvas[py][px] - meanCanvas
						t := target[py][px] - meanTarget
						varCanvas += c * c
						varTarget += t * t
						covar += c * t
					}
				}
			}

			varCanvas /= float64(n)
			varTarget /= float64(n)
			covar /= float64(n)

			// SSIM formula
			numerator := (2*meanCanvas*meanTarget + c1) * (2*covar + c2)
			denominator := (meanCanvas*meanCanvas + meanTarget*meanTarget + c1) *
				(varCanvas + varTarget + c2)

			if denominator > 0 {
				ssimSum += numerator / denominator
				count++
			}
		}
	}

	if count > 0 {
		return ssimSum / float64(count)
	}
	return 0.0
}

func calculateMeanBrightnessV31(canvas [][]float64, width, height int) float64 {
	sum := 0.0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum += canvas[y][x]
		}
	}
	return sum / float64(width*height)
}
