package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// CLI flags
	inputPath := flag.String("input", "", "Input image path (required)")
	outputPath := flag.String("output", "", "Output SVG path (default: auto-generated)")
	numPins := flag.Int("pins", 200, "Number of pins around the circle")
	numLines := flag.Int("lines", 2000, "Number of lines to draw")
	lineWeight := flag.Int("weight", 30, "Line weight (darkness contribution)")
	minDistance := flag.Int("min-dist", 20, "Minimum distance between consecutive pins")
	workers := flag.Int("workers", 8, "Number of parallel workers for line evaluation")
	
	flag.Parse()

	if *inputPath == "" {
		fmt.Println("Error: --input is required")
		flag.Usage()
		os.Exit(1)
	}

	// Auto-generate output path if not provided
	if *outputPath == "" {
		baseName := filepath.Base(*inputPath)
		ext := filepath.Ext(baseName)
		nameWithoutExt := baseName[:len(baseName)-len(ext)]
		timestamp := time.Now().Format("20060102_150405")
		*outputPath = fmt.Sprintf("string_art_output/%s_%s.svg", nameWithoutExt, timestamp)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	fmt.Printf("String Art Generator (Go)\n")
	fmt.Printf("=========================\n")
	fmt.Printf("Input:        %s\n", *inputPath)
	fmt.Printf("Output:       %s\n", *outputPath)
	fmt.Printf("Pins:         %d\n", *numPins)
	fmt.Printf("Lines:        %d\n", *numLines)
	fmt.Printf("Line Weight:  %d\n", *lineWeight)
	fmt.Printf("Min Distance: %d\n", *minDistance)
	fmt.Printf("Workers:      %d\n", *workers)
	fmt.Printf("\n")

	startTime := time.Now()

	// Load and preprocess image
	fmt.Println("Loading image...")
	img, err := LoadImage(*inputPath)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	fmt.Println("Preprocessing image (edge detection)...")
	processed := PreprocessImage(img)

	// Generate string art
	fmt.Println("Generating string art...")
	config := &Config{
		NumPins:     *numPins,
		NumLines:    *numLines,
		LineWeight:  *lineWeight,
		MinDistance: *minDistance,
		Workers:     *workers,
	}

	lines := GenerateStringArt(processed, config)

	// Export to SVG
	fmt.Println("Exporting to SVG...")
	if err := ExportSVG(lines, config, *outputPath); err != nil {
		log.Fatalf("Failed to export SVG: %v", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n✓ Done in %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Output saved to: %s\n", *outputPath)
}
