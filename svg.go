package main

import (
	"fmt"
	"os"
)

// ExportSVG exports the string art to SVG format
// MANDATORY RULES (NEVER VIOLATE):
// 1. SVG dimensions: 600mm x 600mm
// 2. Stroke width: calibrated per generator version
// 3. Stroke opacity: 1.0 (fully opaque - physical thread is opaque)
func ExportSVG(lines []Line, config *Config, outputPath string) error {
	const svgWidth = 600.0
	const svgHeight = 600.0
	strokeWidth := 0.18 // mm - default for V10
	if config.Opacity > 0 && config.Opacity < 0.20 {
		// V11 Wu with lower alpha needs thinner SVG strokes to match
		// Calibrated: alpha=0.15 → stroke-width=0.12 gives best SSIM
		strokeWidth = 0.12
	}
	const strokeOpacity = 1.0 // MANDATORY - physical thread is opaque

	// Calculate pin positions
	centerX, centerY := svgWidth/2, svgHeight/2
	radius := (svgWidth / 2) - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	// Create SVG file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write SVG header
	fmt.Fprintf(file, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" 
     width="600mm" 
     height="600mm" 
     viewBox="0 0 600 600">
  <title>String Art v5.0</title>
  <desc>Generated with %d pins and %d lines</desc>
  
  <!-- Background -->
  <rect width="600" height="600" fill="white"/>
  
  <!-- String lines (0.18mm stroke, fully opaque - physical thread) -->
  <g id="strings" stroke="black" stroke-width="%.2f" stroke-opacity="%.1f" fill="none">
`, config.NumPins, len(lines), strokeWidth, strokeOpacity)

	// Write lines
	for _, line := range lines {
		from := pins[line.From]
		to := pins[line.To]
		fmt.Fprintf(file, `    <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f"/>
`, from.X, from.Y, to.X, to.Y)
	}

	// Close SVG
	fmt.Fprintf(file, `  </g>
</svg>
`)

	return nil
}

// ExportSVGDual exports dual-color string art to SVG format
func ExportSVGDual(blackLines, whiteLines []Line, config *Config, outputPath string) error {
	const svgWidth = 600.0
	const svgHeight = 600.0
	const strokeWidth = 0.18
	const strokeOpacity = 1.0

	centerX, centerY := svgWidth/2, svgHeight/2
	radius := (svgWidth / 2) - 10
	pins := GeneratePins(config.NumPins, radius, centerX, centerY)

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" 
     width="600mm" 
     height="600mm" 
     viewBox="0 0 600 600">
  <title>Dual-Color String Art v5.0</title>
  <desc>Generated with %d pins, %d black lines, %d white lines</desc>
  
  <!-- Background -->
  <rect width="600" height="600" fill="white" fill-opacity="0.5"/>
  
  <!-- Black string lines -->
  <g id="black-strings" stroke="black" stroke-width="%.2f" stroke-opacity="%.1f" fill="none">
`, config.NumPins, len(blackLines), len(whiteLines), strokeWidth, strokeOpacity)

	for _, line := range blackLines {
		from := pins[line.From]
		to := pins[line.To]
		fmt.Fprintf(file, `    <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f"/>
`, from.X, from.Y, to.X, to.Y)
	}

	fmt.Fprintf(file, `  </g>
  
  <!-- White string lines -->
  <g id="white-strings" stroke="white" stroke-width="%.2f" stroke-opacity="%.1f" fill="none">
`, strokeWidth, strokeOpacity)

	for _, line := range whiteLines {
		from := pins[line.From]
		to := pins[line.To]
		fmt.Fprintf(file, `    <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f"/>
`, from.X, from.Y, to.X, to.Y)
	}

	fmt.Fprintf(file, `  </g>
</svg>
`)

	return nil
}
