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
	edgeWeight := flag.Float64("edge-weight", 2.0, "Edge detection multiplier (prioritize edges, default 2.0)")
	
	// v2.1.0 parameters
	opacity := flag.Float64("opacity", 1.0, "String opacity (0.0-1.0, default 1.0)")
	randomSampling := flag.Bool("random-sampling", false, "Enable random sampling optimization")
	sampleSize := flag.Int("sample-size", 1000, "Number of pins to sample per iteration (with --random-sampling)")
	
	// v2.2.0 parameters
	adaptiveStop := flag.Bool("adaptive-stop", true, "Enable adaptive stopping (quality plateau detection)")
	stopThreshold := flag.Float64("stop-threshold", 0.5, "Quality plateau threshold for adaptive stopping")
	lookAhead := flag.Bool("look-ahead", false, "Enable look-ahead optimization (slower, better quality)")
	
	// v4.0.0 parameters (dual-color)
	dualColor := flag.Bool("dual-color", false, "Enable dual-color mode (black + white threads)")
	
	// v5.0.0 parameters (simulated annealing)
	useSimulatedAnnealing := flag.Bool("sa", false, "Use Simulated Annealing algorithm (slower, better quality)")
	
	flag.Parse()

	if *inputPath == "" {
		fmt.Println("Error: --input is required")
		flag.Usage()
		os.Exit(1)
	}

	// MANDATORY RULE: Pin maximal 360 (1 pin per degree)
	if *numPins > 360 {
		fmt.Printf("Error: --pins cannot exceed 360 (requested: %d)\n", *numPins)
		fmt.Println("Reason: Maximum 1 pin per degree for physical construction")
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
	processed, edgeMap := PreprocessImage(img)

	// Generate string art
	fmt.Println("Generating string art...")
	config := &Config{
		NumPins:        *numPins,
		NumLines:       *numLines,
		LineWeight:     *lineWeight,
		MinDistance:    *minDistance,
		Workers:        *workers,
		EdgeWeight:     *edgeWeight,
		Opacity:        *opacity,
		RandomSampling: *randomSampling,
		SampleSize:     *sampleSize,
		AdaptiveStop:   *adaptiveStop,
		StopThreshold:  *stopThreshold,
		LookAhead:      *lookAhead,
	}

	var canvas [][]int
	
	if *useSimulatedAnnealing {
		// Simulated Annealing mode
		fmt.Println("Mode: SIMULATED ANNEALING (global optimization)")
		lines, canvasSA := GenerateStringArtSA(processed, edgeMap, config)
		canvas = canvasSA
		
		// Export to SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}
	} else if *dualColor {
		// Dual-color mode: black + white threads
		fmt.Println("Mode: DUAL-COLOR (black + white threads)")
		
		// Load image with alpha channel
		imgRGBA, err := LoadImageRGBA(*inputPath)
		if err != nil {
			log.Fatalf("Failed to load RGBA image: %v", err)
		}
		
		// Preprocess with alpha awareness
		processedRGBA, edgeMapRGBA := PreprocessImageRGBA(imgRGBA)
		
		blackLines, whiteLines, canvasDual := GenerateStringArtDual(processedRGBA, edgeMapRGBA, config)
		canvas = canvasDual
		
		// Export dual-color SVG
		fmt.Println("Exporting dual-color SVG...")
		if err := ExportSVGDual(blackLines, whiteLines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}
	} else {
		// Single-color mode (original)
		fmt.Println("Mode: SINGLE-COLOR (black threads only)")
		lines, canvasSingle := GenerateStringArt(processed, edgeMap, config)
		canvas = canvasSingle
		
		// Export to SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}
	}

	// Render canvas state to PNG (for showcase - this is what Python does)
	canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
	fmt.Println("Rendering canvas state to PNG...")
	if err := RenderCanvasToImage(canvas, canvasPngPath); err != nil {
		log.Fatalf("Failed to render canvas: %v", err)
	}
	fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)

	elapsed := time.Since(startTime)
	fmt.Printf("\n✓ Done in %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Output saved to: %s\n", *outputPath)
}
