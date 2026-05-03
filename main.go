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
	numPins := flag.Int("pins", 300, "Number of pins around the circle (default: 300)")
	numLines := flag.Int("lines", 2500, "Number of lines to draw (default: 2500)")
	lineWeight := flag.Int("weight", 20, "Line weight (darkness contribution, default: 20)")
	minDistance := flag.Int("min-dist", 15, "Minimum distance between consecutive pins (default: 15)")
	workers := flag.Int("workers", 8, "Number of parallel workers for line evaluation")
	edgeWeight := flag.Float64("edge-weight", 3.0, "Edge detection multiplier (default 3.0)")

	// Optional parameters
	opacity := flag.Float64("opacity", 1.0, "String opacity (0.0-1.0, default 1.0)")
	randomSampling := flag.Bool("random-sampling", false, "Enable random sampling optimization")
	sampleSize := flag.Int("sample-size", 1000, "Number of pins to sample per iteration (with --random-sampling)")
	adaptiveStop := flag.Bool("adaptive-stop", true, "Enable adaptive stopping (quality plateau detection)")
	stopThreshold := flag.Float64("stop-threshold", 0.5, "Quality plateau threshold for adaptive stopping")
	lookAhead := flag.Bool("look-ahead", false, "Enable look-ahead optimization (slower, better quality)")

	// Enhanced mode
	enhanced := flag.Bool("enhanced", false, "Use V10.1 Enhanced generator (multi-start, iterative replacement)")
	wuMode := flag.Bool("wu", false, "Use V11 Wu anti-aliased generator (better SVG match)")

	// Legacy modes
	legacyMode := flag.Bool("legacy", false, "Use legacy v3.x generator")
	dualColor := flag.Bool("dual-color", false, "Enable dual-color mode (black + white threads)")
	useSimulatedAnnealing := flag.Bool("sa", false, "Use Simulated Annealing algorithm (slower)")
	highContrast := flag.Bool("high-contrast", false, "Focus on high-contrast areas only")

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

	// Auto-generate output path if not specified
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

	fmt.Printf("String Art Generator v10.0 (Source-Over)\n")
	fmt.Printf("=====================================================\n")
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
	imgRaw, err := LoadImage(*inputPath)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	fmt.Println("Preprocessing image (edge detection + contrast enhancement)...")
	processed, edgeMap := PreprocessImage(imgRaw)

	// Generate string art
	fmt.Println("Generating string art...")
	config := &Config{
		NumPins:        *numPins,
		NumLines:       *numLines,
		LineWeight:     *lineWeight,
		MinDistance:    *minDistance,
		Workers:       *workers,
		EdgeWeight:    *edgeWeight,
		Opacity:       *opacity,
		RandomSampling: *randomSampling,
		SampleSize:    *sampleSize,
		AdaptiveStop:  *adaptiveStop,
		StopThreshold: *stopThreshold,
		LookAhead:     *lookAhead,
	}

	if *legacyMode {
		// Legacy modes
		if *highContrast {
			fmt.Println("Mode: LEGACY HIGH-CONTRAST")
			lines, canvasHC := GenerateStringArtHighContrast(processed, edgeMap, config)
			if err := ExportSVG(lines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
			canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
			if err := RenderCanvasToImage(canvasHC, canvasPngPath); err != nil {
				log.Fatalf("Failed to render canvas: %v", err)
			}
			fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
		} else if *useSimulatedAnnealing {
			fmt.Println("Mode: LEGACY SIMULATED ANNEALING")
			lines, canvasSA := GenerateStringArtSA(processed, edgeMap, config)
			if err := ExportSVG(lines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
			canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
			if err := RenderCanvasToImage(canvasSA, canvasPngPath); err != nil {
				log.Fatalf("Failed to render canvas: %v", err)
			}
			fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
		} else if *dualColor {
			fmt.Println("Mode: LEGACY DUAL-COLOR")
			imgRGBA, err := LoadImageRGBA(*inputPath)
			if err != nil {
				log.Fatalf("Failed to load RGBA image: %v", err)
			}
			processedRGBA, edgeMapRGBA := PreprocessImageRGBA(imgRGBA)
			blackLines, whiteLines, canvasDual := GenerateStringArtDual(processedRGBA, edgeMapRGBA, config)
			if err := ExportSVGDual(blackLines, whiteLines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
			canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
			if err := RenderCanvasToImage(canvasDual, canvasPngPath); err != nil {
				log.Fatalf("Failed to render canvas: %v", err)
			}
			fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
		} else {
			fmt.Println("Mode: LEGACY SINGLE-COLOR")
			lines, canvasSingle := GenerateStringArt(processed, edgeMap, config)
			if err := ExportSVG(lines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
			canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
			if err := RenderCanvasToImage(canvasSingle, canvasPngPath); err != nil {
				log.Fatalf("Failed to render canvas: %v", err)
			}
			fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
		}
	} else if *wuMode {
		// V11 Wu Anti-Aliased (better SVG match)
		fmt.Println("Mode: V11.0 Wu Anti-Aliased (better SVG match)")
		lines, canvasF := GenerateStringArtV11(processed, edgeMap, config)

		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasToImageFloat(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *enhanced {
		// V10.1 Enhanced (multi-start, iterative replacement, adaptive penalties)
		fmt.Println("Mode: V10.1 Enhanced (multi-start, iterative replacement)")
		lines, canvasF := GenerateStringArtV10Enhanced(processed, edgeMap, config)

		// Export SVG
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Export canvas PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasToImageFloat(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else {
		// DEFAULT: V10.0 Source-Over (BEST VERSION - confirmed by metrics + vision AI)
		fmt.Println("Mode: V10.0 Source-Over (BEST - alpha=0.25, Bresenham)")
		lines, canvasF := GenerateStringArtV10(processed, edgeMap, config)

		// Export SVG
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Export canvas PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasToImageFloat(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)

		// Also render SVG to PNG for comparison
		mobilePngPath := (*outputPath)[:len(*outputPath)-4] + "_mobile_400px.png"
		fmt.Printf("\nRendering SVG to PNG for comparison...\n")
		renderCmd := fmt.Sprintf("rsvg-convert -w 400 -h 400 %s -o %s 2>/dev/null", *outputPath, mobilePngPath)
		_ = renderCmd // Will be done externally
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nGeneration complete in %.1f seconds\n", elapsed.Seconds())
	fmt.Printf("SVG output: %s\n", *outputPath)

	// Render hint
	mobilePngPath := (*outputPath)[:len(*outputPath)-4] + "_mobile_400px.png"
	fmt.Printf("\nTo render SVG to PNG preview (accurate mobile view):\n")
	fmt.Printf("  rsvg-convert -w 400 -h 400 %s -o %s\n", *outputPath, mobilePngPath)
}
