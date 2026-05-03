package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Bresenham's line algorithm (current v10 implementation)
func drawLineBresenham(img *image.Gray, x0, y0, x1, y1 int, alpha uint8) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		// Blend pixel with alpha
		old := img.GrayAt(x0, y0).Y
		new := uint8(int(old) - int(alpha)*int(255-old)/255)
		img.SetGray(x0, y0, color.Gray{Y: new})

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

// Xiaolin Wu's line algorithm (anti-aliased)
func drawLineWu(img *image.Gray, x0f, y0f, x1f, y1f float64, alpha uint8) {
	steep := math.Abs(y1f-y0f) > math.Abs(x1f-x0f)

	if steep {
		x0f, y0f = y0f, x0f
		x1f, y1f = y1f, x1f
	}
	if x0f > x1f {
		x0f, x1f = x1f, x0f
		y0f, y1f = y1f, y0f
	}

	dx := x1f - x0f
	dy := y1f - y0f
	gradient := dy / dx
	if dx == 0 {
		gradient = 1.0
	}

	// Handle first endpoint
	xend := math.Round(x0f)
	yend := y0f + gradient*(xend-x0f)
	xgap := 1.0 - frac(x0f+0.5)
	xpxl1 := int(xend)
	ypxl1 := int(yend)

	if steep {
		plot(img, ypxl1, xpxl1, (1.0-frac(yend))*xgap, alpha)
		plot(img, ypxl1+1, xpxl1, frac(yend)*xgap, alpha)
	} else {
		plot(img, xpxl1, ypxl1, (1.0-frac(yend))*xgap, alpha)
		plot(img, xpxl1, ypxl1+1, frac(yend)*xgap, alpha)
	}

	intery := yend + gradient

	// Handle second endpoint
	xend = math.Round(x1f)
	yend = y1f + gradient*(xend-x1f)
	xgap = frac(x1f + 0.5)
	xpxl2 := int(xend)
	ypxl2 := int(yend)

	if steep {
		plot(img, ypxl2, xpxl2, (1.0-frac(yend))*xgap, alpha)
		plot(img, ypxl2+1, xpxl2, frac(yend)*xgap, alpha)
	} else {
		plot(img, xpxl2, ypxl2, (1.0-frac(yend))*xgap, alpha)
		plot(img, xpxl2, ypxl2+1, frac(yend)*xgap, alpha)
	}

	// Main loop
	if steep {
		for x := xpxl1 + 1; x < xpxl2; x++ {
			plot(img, int(intery), x, 1.0-frac(intery), alpha)
			plot(img, int(intery)+1, x, frac(intery), alpha)
			intery += gradient
		}
	} else {
		for x := xpxl1 + 1; x < xpxl2; x++ {
			plot(img, x, int(intery), 1.0-frac(intery), alpha)
			plot(img, x, int(intery)+1, frac(intery), alpha)
			intery += gradient
		}
	}
}

func plot(img *image.Gray, x, y int, brightness float64, alpha uint8) {
	bounds := img.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return
	}

	old := img.GrayAt(x, y).Y
	effectiveAlpha := float64(alpha) * brightness / 255.0
	new := float64(old) * (1.0 - effectiveAlpha)
	img.SetGray(x, y, color.Gray{Y: uint8(new)})
}

func frac(x float64) float64 {
	return x - math.Floor(x)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	width, height := 400, 400

	// Create two test images
	imgBresenham := image.NewGray(image.Rect(0, 0, width, height))
	imgWu := image.NewGray(image.Rect(0, 0, width, height))

	// Fill with white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			imgBresenham.SetGray(x, y, color.Gray{Y: 255})
			imgWu.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	// Draw test lines at various angles
	alpha := uint8(64) // 25% opacity like v10

	// Horizontal line
	drawLineBresenham(imgBresenham, 50, 50, 350, 50, alpha)
	drawLineWu(imgWu, 50, 50, 350, 50, alpha)

	// Vertical line
	drawLineBresenham(imgBresenham, 50, 100, 50, 350, alpha)
	drawLineWu(imgWu, 50, 100, 50, 350, alpha)

	// Diagonal 45°
	drawLineBresenham(imgBresenham, 100, 100, 300, 300, alpha)
	drawLineWu(imgWu, 100, 100, 300, 300, alpha)

	// Shallow angle
	drawLineBresenham(imgBresenham, 100, 150, 350, 200, alpha)
	drawLineWu(imgWu, 100, 150, 350, 200, alpha)

	// Steep angle
	drawLineBresenham(imgBresenham, 150, 100, 200, 350, alpha)
	drawLineWu(imgWu, 150, 100, 200, 350, alpha)

	// Save images
	fBresenham, _ := os.Create("test_bresenham.png")
	png.Encode(fBresenham, imgBresenham)
	fBresenham.Close()

	fWu, _ := os.Create("test_wu.png")
	png.Encode(fWu, imgWu)
	fWu.Close()

	fmt.Println("✅ Generated test_bresenham.png and test_wu.png")
	fmt.Println("📊 Visual comparison:")
	fmt.Println("   - Bresenham: Hard edges, aliased")
	fmt.Println("   - Wu: Smooth edges, anti-aliased")
	fmt.Println("🎯 Wu algorithm should better match SVG rendering")
}
