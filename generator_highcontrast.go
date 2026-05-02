package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// GenerateStringArtHighContrast focuses on high-contrast areas only
// Better for recognizable subjects
func GenerateStringArtHighContrast(img *image.Gray, edgeMap *image.Gray, config *Config) ([]Line, [][]int) {
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

	fmt.Println("=== High-Contrast String Art ===")
	
	// Create importance map (focus on dark areas + edges)
	importanceMap := createImportanceMap(img, edgeMap)
	
	lines := make([]Line, 0, config.NumLines)
	currentPin := 0
	
	for i := 0; i < config.NumLines; i++ {
		bestLine := findBestLineHighContrast(currentPin, pins, canvas, img, edgeMap, importanceMap, config)
		
		if bestLine.Score <= 0 {
			break
		}
		
		// Apply to canvas
		effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
		drawLine(canvas, pins[bestLine.From], pins[bestLine.To], effectiveWeight)
		
		lines = append(lines, bestLine)
		currentPin = bestLine.To
		
		if (i+1)%100 == 0 {
			fmt.Printf("Progress: %d/%d lines (score: %.2f)\n", i+1, config.NumLines, bestLine.Score)
		}
	}
	
	fmt.Printf("Generated %d lines\n", len(lines))
	return lines, canvas
}

// createImportanceMap creates a map highlighting important areas
func createImportanceMap(img *image.Gray, edgeMap *image.Gray) *image.Gray {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	importance := image.NewGray(bounds)
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grayVal := img.GrayAt(x, y).Y
			edgeVal := edgeMap.GrayAt(x, y).Y
			
			// High importance for:
			// 1. Very dark areas (< 100)
			// 2. Strong edges (> 50)
			var importanceVal uint8
			
			if grayVal < 100 {
				// Dark area - high importance
				importanceVal = 255 - grayVal
			} else if edgeVal > 50 {
				// Edge - medium importance
				importanceVal = edgeVal
			} else {
				// Light area - low importance
				importanceVal = 0
			}
			
			importance.SetGray(x, y, color.Gray{Y: importanceVal})
		}
	}
	
	return importance
}

// findBestLineHighContrast finds best line focusing on important areas
func findBestLineHighContrast(fromPin int, pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, importanceMap *image.Gray, config *Config) Line {
	numPins := len(pins)
	
	bestScore := -1e9
	bestToPin := -1
	
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
		
		score := evaluateLineHighContrast(fromPin, toPin, pins, canvas, img, edgeMap, importanceMap, config)
		
		if score > bestScore {
			bestScore = score
			bestToPin = toPin
		}
	}
	
	return Line{
		From:  fromPin,
		To:    bestToPin,
		Score: bestScore,
	}
}

// evaluateLineHighContrast evaluates line with importance weighting
func evaluateLineHighContrast(fromPin, toPin int, pins []Pin, canvas [][]int, img *image.Gray, edgeMap *image.Gray, importanceMap *image.Gray, config *Config) float64 {
	from := pins[fromPin]
	to := pins[toPin]
	
	pixels := GetPixelsOnLine(int(from.X), int(from.Y), int(to.X), int(to.Y))
	
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	score := 0.0
	totalImportance := 0.0
	
	effectiveWeight := int(float64(config.LineWeight) * config.Opacity)
	
	for _, p := range pixels {
		x, y := p[0], p[1]
		
		if x < 0 || x >= width || y < 0 || y >= height {
			continue
		}
		
		// Get importance
		importance := float64(importanceMap.GrayAt(x, y).Y)
		
		// Skip unimportant areas
		if importance < 10 {
			continue
		}
		
		totalImportance += importance
		
		// Calculate improvement
		currentCanvas := canvas[y][x]
		targetGray := img.GrayAt(x, y).Y
		
		newCanvas := currentCanvas - effectiveWeight
		if newCanvas < 0 {
			newCanvas = 0
		}
		
		oldError := float64(abs(int(targetGray) - currentCanvas))
		newError := float64(abs(int(targetGray) - newCanvas))
		improvement := oldError - newError
		
		// Weight by importance
		score += improvement * importance
	}
	
	// Normalize by total importance
	if totalImportance > 0 {
		score /= totalImportance
	}
	
	return score
}
