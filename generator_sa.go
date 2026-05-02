package main

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"time"
)

// GenerateStringArtSA generates string art using Simulated Annealing
func GenerateStringArtSA(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]int) {
	rand.Seed(time.Now().UnixNano())
	
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	// Create canvas
	canvas := make([][]int, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]int, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 255
		}
	}

	// Generate pins
	centerX, centerY := float64(width)/2, float64(height)/2
	radius := math.Min(centerX, centerY) - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	fmt.Println("=== Simulated Annealing String Art ===")
	
	// Phase 1: Greedy initialization (fast, get to ~70% quality)
	fmt.Println("Phase 1: Greedy initialization...")
	lines := greedyInitialization(pins, canvas, img, edgeMap, config)
	
	// Phase 2: Simulated Annealing refinement
	fmt.Println("\nPhase 2: Simulated Annealing refinement...")
	lines = simulatedAnnealingRefinement(lines, pins, canvas, img, edgeMap, config)
	
	// Rebuild canvas from final lines
	canvas = rebuildCanvas(lines, pins, width, height, config)
	
	fmt.Printf("\nFinal: %d lines\n", len(lines))
	return lines, canvas
}

// greedyInitialization: fast greedy to get initial solution
func greedyInitialization(pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, config *Config) []Line {
	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	
	// Use 70% of target lines for initialization
	initLines := int(float64(config.NumLines) * 0.7)
	
	for i := 0; i < initLines; i++ {
		bestLine := findBestLine(currentPin, pins, canvas, img, edgeMap, config)
		
		if bestLine.Score <= 0 {
			break
		}
		
		// Apply to canvas
		effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
		drawLine(canvas, pins[bestLine.From], pins[bestLine.To], effectiveWeight)
		
		lines = append(lines, bestLine)
		currentPin = bestLine.To
		
		if (i+1)%100 == 0 {
			fmt.Printf("  Init: %d/%d lines (score: %.2f)\n", i+1, initLines, bestLine.Score)
		}
	}
	
	fmt.Printf("  Initialized with %d lines\n", len(lines))
	return lines
}

// simulatedAnnealingRefinement: refine solution with SA
func simulatedAnnealingRefinement(lines []Line, pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, config *Config) []Line {
	// SA parameters
	initialTemp := 100.0
	finalTemp := 0.1
	coolingRate := 0.995
	iterationsPerTemp := 50
	
	currentLines := make([]Line, len(lines))
	copy(currentLines, lines)
	
	bestLines := make([]Line, len(lines))
	copy(bestLines, lines)
	
	// Calculate initial energy (error)
	currentEnergy := calculateEnergy(currentLines, pins, img, edgeMap, config)
	bestEnergy := currentEnergy
	
	temperature := initialTemp
	iteration := 0
	acceptedMoves := 0
	
	fmt.Printf("  Initial energy: %.2f\n", currentEnergy)
	
	for temperature > finalTemp {
		for i := 0; i < iterationsPerTemp; i++ {
			iteration++
			
			// Generate neighbor solution
			neighborLines := generateNeighbor(currentLines, pins, config)
			neighborEnergy := calculateEnergy(neighborLines, pins, img, edgeMap, config)
			
			// Calculate acceptance probability
			deltaE := neighborEnergy - currentEnergy
			acceptProb := 1.0
			if deltaE > 0 {
				acceptProb = math.Exp(-deltaE / temperature)
			}
			
			// Accept or reject
			if rand.Float64() < acceptProb {
				currentLines = neighborLines
				currentEnergy = neighborEnergy
				acceptedMoves++
				
				// Update best if better
				if currentEnergy < bestEnergy {
					bestLines = make([]Line, len(currentLines))
					copy(bestLines, currentLines)
					bestEnergy = currentEnergy
					fmt.Printf("  Iter %d: New best energy: %.2f (temp: %.2f)\n", 
						iteration, bestEnergy, temperature)
				}
			}
		}
		
		temperature *= coolingRate
		
		if iteration%500 == 0 {
			acceptRate := float64(acceptedMoves) / float64(iteration) * 100
			fmt.Printf("  Iter %d: temp=%.2f, energy=%.2f, accept=%.1f%%\n", 
				iteration, temperature, currentEnergy, acceptRate)
		}
	}
	
	fmt.Printf("  SA complete: %d iterations, %.1f%% accepted\n", 
		iteration, float64(acceptedMoves)/float64(iteration)*100)
	fmt.Printf("  Best energy: %.2f (improved by %.2f)\n", 
		bestEnergy, calculateEnergy(lines, pins, img, edgeMap, config)-bestEnergy)
	
	return bestLines
}

// generateNeighbor: create neighbor solution by modifying current solution
func generateNeighbor(lines []Line, pins []Pin, config *Config) []Line {
	neighbor := make([]Line, len(lines))
	copy(neighbor, lines)
	
	// Random modification strategies
	strategy := rand.Intn(3)
	
	switch strategy {
	case 0:
		// Swap two random lines
		if len(lines) >= 2 {
			i := rand.Intn(len(lines))
			j := rand.Intn(len(lines))
			neighbor[i], neighbor[j] = neighbor[j], neighbor[i]
		}
		
	case 1:
		// Replace random line with new random line
		if len(lines) > 0 {
			idx := rand.Intn(len(lines))
			fromPin := rand.Intn(len(pins))
			toPin := rand.Intn(len(pins))
			
			// Ensure valid connection
			distance := abs(toPin - fromPin)
			if distance >= config.MinDistance && distance <= len(pins)-config.MinDistance {
				neighbor[idx] = Line{
					From:  fromPin,
					To:    toPin,
					Score: 0, // Will be recalculated
				}
			}
		}
		
	case 2:
		// Reverse a subsequence
		if len(lines) >= 2 {
			start := rand.Intn(len(lines) - 1)
			end := start + 1 + rand.Intn(min(10, len(lines)-start-1))
			
			// Reverse lines[start:end]
			for i, j := start, end-1; i < j; i, j = i+1, j-1 {
				neighbor[i], neighbor[j] = neighbor[j], neighbor[i]
			}
		}
	}
	
	return neighbor
}

// calculateEnergy: calculate total error (lower is better)
func calculateEnergy(lines []Line, pins []Pin, img *image.Gray, edgeMap *image.Gray, config *Config) float64 {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	// Rebuild canvas from lines
	canvas := make([][]int, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]int, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 255
		}
	}
	
	effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
	for _, line := range lines {
		drawLine(canvas, pins[line.From], pins[line.To], effectiveWeight)
	}
	
	// Calculate total error with edge weighting
	totalError := 0.0
	pixelCount := 0
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			targetGray := img.GrayAt(x, y).Y
			canvasGray := canvas[y][x]
			edgeValue := float64(edgeMap.GrayAt(x, y).Y)
			
			// Error with edge weighting
			error := float64(abs(int(targetGray) - canvasGray))
			edgeWeight := 1.0 + (edgeValue/255.0)*config.EdgeWeight
			
			totalError += error * edgeWeight
			pixelCount++
		}
	}
	
	return totalError / float64(pixelCount)
}

// rebuildCanvas: rebuild canvas from final line sequence
func rebuildCanvas(lines []Line, pins []Pin, width, height int, config *Config) [][]int {
	canvas := make([][]int, height)
	for y := 0; y < height; y++ {
		canvas[y] = make([]int, width)
		for x := 0; x < width; x++ {
			canvas[y][x] = 255
		}
	}
	
	effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
	for _, line := range lines {
		drawLine(canvas, pins[line.From], pins[line.To], effectiveWeight)
	}
	
	return canvas
}
