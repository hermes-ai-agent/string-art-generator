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
	numLines := flag.Int("lines", 4000, "Number of lines to draw (default: 4000)")
	lineWeight := flag.Int("weight", 20, "Line weight (darkness contribution, default: 20)")
	minDistance := flag.Int("min-dist", 15, "Minimum distance between consecutive pins (default: 15)")
	workers := flag.Int("workers", 8, "Number of parallel workers for line evaluation")
	edgeWeight := flag.Float64("edge-weight", 3.0, "Edge detection multiplier (default 3.0)")
	
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
	
	// v3.8.0 parameters (high-contrast)
	highContrast := flag.Bool("high-contrast", false, "Focus on high-contrast areas only (better for recognizable subjects)")

	// v5.0.0 parameters (legacy mode)
	legacyMode := flag.Bool("legacy", false, "Use legacy v3.x generator instead of v5.0")
	
	// v6.0.0 parameters (supersampled rendering)
	useSupersampled := flag.Bool("v6", false, "Use v6.0 supersampled rendering (Birsak 2018 style)")

	// v7.0.0 parameters (SSIM-optimized)
	useSSIMOptimized := flag.Bool("v7", false, "Use v7.0 SSIM-optimized supersampled rendering")

	// v8.0.0 parameters (enhanced supersampled)
	useEnhancedSupersampled := flag.Bool("v8", false, "Use v8.0 enhanced supersampled rendering with add/remove optimization")

	// v8.1.0 parameters (enhanced v5-based)
	useV8Enhanced := flag.Bool("v8e", false, "Use v8.0 enhanced v5-based rendering with face detection and add/remove optimization")

	// v9.0.0 parameters (Birsak 2018 supersampled)
	useV9Birsak := flag.Bool("v9", false, "Use v9.0 Birsak 2018 supersampled rendering (4x supersampling)")

	// v10.0.0 parameters (Enhanced perceptual)
	useV10Enhanced := flag.Bool("v10", false, "Use v10.0 Enhanced perceptual rendering with sophisticated face detection")

	// v11.0.0 parameters (Improved SSIM-focused)
	useV11Improved := flag.Bool("v11", false, "Use v11.0 Improved SSIM-focused rendering with 2x supersampling and enhanced optimization")

	// v12.0.0 parameters (Focused improvements)
	useV12Focused := flag.Bool("v12", false, "Use v12.0 Focused improvements with enhanced face detection and better scoring")

	// v13.0.0 parameters (Optimized V5 improvements)
	useV13Optimized := flag.Bool("v13", false, "Use v13.0 Optimized improvements over V5 with better mobile rendering calibration")

	// v14.0.0 parameters (Birsak 2018 supersampled)
	useV14Birsak := flag.Bool("v14", false, "Use v14.0 Birsak 2018 supersampled rendering (4x supersampling with enhanced optimization)")

	// v14.1.0 parameters (Enhanced focused improvements)
	useV14Enhanced := flag.Bool("v14e", false, "Use v14.1 Enhanced focused improvements (better face detection, multi-pass removal, calibrated alpha)")

	// v14.2.0 parameters (Fast focused improvements)
	useV14Fast := flag.Bool("v14f", false, "Use v14.2 Fast focused improvements (optimized face detection, single-pass removal, calibrated alpha)")

	// v14.3.0 parameters (Calibrated improvements)
	useV14Calibrated := flag.Bool("v14c", false, "Use v14.3 Calibrated improvements (precise mobile SVG calibration, enhanced face detection)")

	// v16.0.0 parameters (Fixed improvements)
	useV16Fixed := flag.Bool("v16", false, "Use v16.0 Fixed improvements (corrected v15 with proper SSIM scoring and add/remove optimization)")

	// v17.0.0 parameters (Enhanced improvements)
	useV17Enhanced := flag.Bool("v17", false, "Use v17.0 Enhanced improvements (4x supersampling, MSE-based scoring, enhanced face detection)")

	// v18.0.0 parameters (Birsak 2018 improvements)
	useV18Birsak := flag.Bool("v18", false, "Use v18.0 Birsak 2018 improvements (8x supersampling, perceptual scoring, calibrated alpha)")

	// v19.0.0 parameters (Focused improvements)
	useV19Focused := flag.Bool("v19", false, "Use v19.0 Focused improvements (enhanced face detection, SSIM-based add/remove, 3-phase optimization)")

	// v20.0.0 parameters (Improved v5)
	useV20Improved := flag.Bool("v20", false, "Use v20.0 Improved v5 (enhanced face detection, SSIM-based optimization, 3-phase processing)")
	
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

	fmt.Printf("String Art Generator v5.0 (Go)\n")
	fmt.Printf("==============================\n")
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
		Workers:        *workers,
		EdgeWeight:     *edgeWeight,
		Opacity:        *opacity,
		RandomSampling: *randomSampling,
		SampleSize:     *sampleSize,
		AdaptiveStop:   *adaptiveStop,
		StopThreshold:  *stopThreshold,
		LookAhead:      *lookAhead,
	}

	var canvasInt [][]int
	
	if *legacyMode {
		// Legacy modes
		if *highContrast {
			fmt.Println("Mode: LEGACY HIGH-CONTRAST")
			lines, canvasHC := GenerateStringArtHighContrast(processed, edgeMap, config)
			canvasInt = canvasHC
			if err := ExportSVG(lines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
		} else if *useSimulatedAnnealing {
			fmt.Println("Mode: LEGACY SIMULATED ANNEALING")
			lines, canvasSA := GenerateStringArtSA(processed, edgeMap, config)
			canvasInt = canvasSA
			if err := ExportSVG(lines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
		} else if *dualColor {
			fmt.Println("Mode: LEGACY DUAL-COLOR")
			imgRGBA, err := LoadImageRGBA(*inputPath)
			if err != nil {
				log.Fatalf("Failed to load RGBA image: %v", err)
			}
			processedRGBA, edgeMapRGBA := PreprocessImageRGBA(imgRGBA)
			blackLines, whiteLines, canvasDual := GenerateStringArtDual(processedRGBA, edgeMapRGBA, config)
			canvasInt = canvasDual
			if err := ExportSVGDual(blackLines, whiteLines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
		} else {
			fmt.Println("Mode: LEGACY SINGLE-COLOR")
			lines, canvasSingle := GenerateStringArt(processed, edgeMap, config)
			canvasInt = canvasSingle
			if err := ExportSVG(lines, config, *outputPath); err != nil {
				log.Fatalf("Failed to export SVG: %v", err)
			}
		}

		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasToImage(canvasInt, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV8Enhanced {
		// V8.1 mode (enhanced v5-based rendering)
		fmt.Println("Mode: V8.1 (enhanced v5-based rendering with face detection and add/remove optimization)")
		lines, canvasF := GenerateStringArtV8Enhanced(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV14Calibrated {
		// V14.3 mode (Calibrated improvements)
		fmt.Println("Mode: V14.3 (Calibrated improvements with precise mobile SVG calibration, enhanced face detection)")
		lines, canvasF := GenerateStringArtV14Calibrated(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV14Fast {
		// V14.2 mode (Fast focused improvements)
		fmt.Println("Mode: V14.2 (Fast focused improvements with optimized face detection, single-pass removal, calibrated alpha)")
		lines, canvasF := GenerateStringArtV14Fast(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV14Enhanced {
		// V14.1 mode (Enhanced focused improvements)
		fmt.Println("Mode: V14.1 (Enhanced focused improvements with better face detection, multi-pass removal, calibrated alpha)")
		lines, canvasF := GenerateStringArtV14Enhanced(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV14Birsak {
		// V14.0 mode (Birsak 2018 supersampled rendering)
		fmt.Println("Mode: V14.0 (Birsak 2018 supersampled rendering with 8x supersampling and enhanced optimization)")
		lines, canvasF := GenerateStringArtV14Birsak(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV13Optimized {
		// V13.0 mode (Optimized V5 improvements)
		fmt.Println("Mode: V13.0 (Optimized improvements over V5 with better mobile rendering calibration)")
		lines, canvasF := GenerateStringArtV13Optimized(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV12Focused {
		// V12.0 mode (Focused improvements)
		fmt.Println("Mode: V12.0 (Focused improvements with enhanced face detection and better scoring)")
		lines, canvasF := GenerateStringArtV12Focused(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV11Improved {
		// V11.0 mode (Improved SSIM-focused rendering)
		fmt.Println("Mode: V11.0 (Improved SSIM-focused rendering with 2x supersampling and enhanced optimization)")
		lines, canvasF := GenerateStringArtV11Improved(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV10Enhanced {
		// V10.0 mode (Enhanced perceptual rendering)
		fmt.Println("Mode: V10.0 (Enhanced perceptual rendering with sophisticated face detection)")
		lines, canvasF := GenerateStringArtV10Enhanced(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV9Birsak {
		// V9.0 mode (Birsak 2018 supersampled rendering)
		fmt.Println("Mode: V9.0 (Birsak 2018 supersampled rendering - 8x supersampling)")
		lines, canvasF := GenerateStringArtV9Birsak(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useEnhancedSupersampled {
		// V8.0 mode (enhanced supersampled rendering)
		fmt.Println("Mode: V8.0 (enhanced supersampled rendering with add/remove optimization)")
		lines, canvasF := GenerateStringArtV8(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useSSIMOptimized {
		// V7.0 mode (SSIM-optimized)
		fmt.Println("Mode: V7.0 (SSIM-optimized supersampled rendering)")
		lines, canvasF := GenerateStringArtV7(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV20Improved {
		// V20.0 mode (Improved v5)
		fmt.Println("Mode: V20.0 (Improved v5 with enhanced face detection, SSIM-based optimization, 3-phase processing)")
		lines, canvasF := GenerateStringArtV20Improved(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV19Focused {
		// V19.0 mode (Focused improvements)
		fmt.Println("Mode: V19.0 (Focused improvements with enhanced face detection, SSIM-based add/remove, 3-phase optimization)")
		lines, canvasF := GenerateStringArtV19Focused(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV18Birsak {
		// V18.0 mode (Birsak 2018 improvements)
		fmt.Println("Mode: V18.0 (Birsak 2018 improvements with 8x supersampling, perceptual scoring, calibrated alpha)")
		lines, canvasF := GenerateStringArtV18Birsak(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV17Enhanced {
		// V17.0 mode (Enhanced improvements)
		fmt.Println("Mode: V17.0 (Enhanced improvements with 4x supersampling, MSE-based scoring, enhanced face detection)")
		lines, canvasF := GenerateStringArtV17Enhanced(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useV16Fixed {
		// V16.0 mode (Fixed improvements)
		fmt.Println("Mode: V16.0 (Fixed improvements with corrected SSIM scoring and add/remove optimization)")
		lines, canvasF := GenerateStringArtV16Fixed(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else if *useSupersampled {
		// V6.0 mode (supersampled rendering)
		fmt.Println("Mode: V6.0 (supersampled rendering - Birsak 2018 style)")
		lines, canvasF := GenerateStringArtV6(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	} else {
		// V5.0 mode (default)
		fmt.Println("Mode: V5.0 (error reduction + anti-aliasing + fear removal + line removal)")
		lines, canvasF := GenerateStringArtV5(processed, edgeMap, config)

		// Export SVG
		fmt.Println("Exporting to SVG...")
		if err := ExportSVG(lines, config, *outputPath); err != nil {
			log.Fatalf("Failed to export SVG: %v", err)
		}

		// Render canvas to PNG
		canvasPngPath := (*outputPath)[:len(*outputPath)-4] + "_canvas.png"
		if err := RenderCanvasV5ToImage(canvasF, canvasPngPath); err != nil {
			log.Fatalf("Failed to render canvas: %v", err)
		}
		fmt.Printf("Canvas render saved to: %s\n", canvasPngPath)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n✓ Done in %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Output saved to: %s\n", *outputPath)
}
