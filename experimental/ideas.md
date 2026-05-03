# String Art Research Ideas\n\nBaseline: v10.0 (SSIM 0.1959, Quality 7/10) - Source-Over Compositing\nNote: SSIM 0.19-0.20 is NEAR-OPTIMAL for string art (physical limitation)\n\n## Priority Research Areas\n\n1. Fix SVG↔Canvas mismatch\n2. Better importance map\n3. Improved line removal\n4. Multi-resolution approach\n5. Perceptual loss function\n\n---\n\n## Session 2026-05-03 23:00 - Importance Map Enhancement\n\n### Research Topic: Face-Aware Importance Weighting\n\n**Source:** Bridges 2022 paper (Demoussel et al.) + GoCV face detection\n\n**Key Finding:**\nBridges 2022 paper explicitly mentions importance maps as a quality enhancement:\n- \"Grid of scalar values indicating pixel importance\"\n- \"Each potential line's pixel scores multiplied by importance values\"\n- \"Useful for emphasizing salient features (eyes, mouth)\"\n- Current v9 uses uniform weighting (all pixels equal importance)\n\n**Proposed Enhancement:**\n1. Use GoCV's Haar Cascade face detector (gocv.CascadeClassifier)\n2. Detect face regions in input image\n3. Generate importance map:\n   - Face region: 2.0x weight\n   - Eyes/mouth (if detected): 3.0x weight\n   - Background: 1.0x weight\n4. Modify line scoring to multiply pixel differences by importance weights\n\n**Implementation Approach:**\n```go\n// Pseudo-code\nclassifier := gocv.NewCascadeClassifier()\nclassifier.Load(\"haarcascade_frontalface_default.xml\")\nfaces := classifier.DetectMultiScale(inputImage)\n\nimportanceMap := make([][]float64, height, width)\n// Initialize all to 1.0\nfor _, face := range faces {\n    // Set face region to 2.0\n    // Detect eyes within face, set to 3.0\n}\n\n// In line scoring:\nscore := 0.0\nfor each pixel in line {\n    diff := targetPixel - currentPixel\n    score += diff * importanceMap[y][x]\n}\n```\n\n**Expected Impact:**\n- Better facial feature preservation (eyes, nose, mouth)\n- Reduced string waste on background areas\n- Estimated SSIM improvement: +0.02 to +0.05 (based on Bridges paper results)\n- No performance penalty (importance map computed once at start)\n\n**Dependencies:**\n- gocv.io/x/gocv (already in project)\n- Haar cascade XML file (downloadable from OpenCV repo)\n\n**Risk Assessment:**\n- Low risk: Additive feature, doesn't modify core algorithm\n- Fallback: If no faces detected, use uniform weights (current behavior)\n- Testing needed: Portrait vs landscape vs abstract images\n\n**Next Steps (for user decision):**\n1. Download haarcascade_frontalface_default.xml\n2. Implement importance map generator\n3. Modify line scoring in greedy selection\n4. Test on portrait images (Mona Lisa, Einstein, etc.)\n5. Measure SSIM improvement\n\n---\n

## Session 2026-05-04 00:30 - Anti-Aliased Line Rasterization

### Research Topic: Xiaolin Wu's Line Algorithm for SVG Matching

**Source:** 
- Wikipedia: Xiaolin Wu's line algorithm (1991)
- Paper: "An Efficient Antialiasing Technique" (Computer Graphics, July 1991)
- Paper: "Fast Antialiasing" (Dr. Dobb's Journal, June 1992)

**Problem Context:**
Current v10 uses Bresenham's algorithm for line rasterization:
- Canvas-SVG MAE: 35.5 (structural difference from anti-aliasing)
- SVG rendering uses anti-aliasing by default
- Bresenham produces hard-edged pixels (no sub-pixel smoothing)
- This creates a fundamental mismatch between canvas simulation and SVG output

**Key Finding:**
Xiaolin Wu's algorithm provides anti-aliased line drawing by:
1. Drawing **pairs of pixels** instead of single pixels
2. Setting intensities **proportional to distance** from ideal line
3. Handling **sub-pixel positioning** (endpoints don't need integer coords)
4. Using simple arithmetic (suitable for older CPUs/microcontrollers)

**Algorithm Characteristics:**
- **Faster than** naive anti-aliasing approaches
- **Slower than** Bresenham (but provides anti-aliasing)
- **Better match** for SVG rendering behavior
- **Smooth gradients** along line edges

**Implementation Approach:**
```go
// Xiaolin Wu's line algorithm (simplified for 0 ≤ slope ≤ 1)
func drawLineWu(canvas [][]uint8, x0, y0, x1, y1 float64, alpha uint8) {
    dx := x1 - x0
    dy := y1 - y0
    gradient := dy / dx
    
    // Handle first endpoint
    xend := round(x0)
    yend := y0 + gradient * (xend - x0)
    xgap := 1.0 - frac(x0 + 0.5)
    xpxl1 := int(xend)
    ypxl1 := int(yend)
    
    // Draw first endpoint pair
    plot(xpxl1, ypxl1, (1.0 - frac(yend)) * xgap, alpha)
    plot(xpxl1, ypxl1+1, frac(yend) * xgap, alpha)
    
    intery := yend + gradient
    
    // Main loop - draw pixel pairs
    for x := xpxl1 + 1; x < xpxl2; x++ {
        plot(x, int(intery), 1.0 - frac(intery), alpha)
        plot(x, int(intery)+1, frac(intery), alpha)
        intery += gradient
    }
    
    // Handle second endpoint (similar to first)
}

func plot(x, y int, brightness, alpha float64) {
    // Blend with existing pixel using brightness weight
    effectiveAlpha := alpha * brightness
    canvas[y][x] = blend(canvas[y][x], 0, effectiveAlpha)
}

func frac(x float64) float64 {
    return x - math.Floor(x)
}
```

**Expected Impact:**
- **Reduced Canvas-SVG MAE:** From 35.5 to ~15-20 (estimated 40-50% reduction)
- **Better visual match:** Canvas preview will look more like final SVG
- **SSIM impact:** Minimal (±0.01) - SSIM measures structural similarity, not anti-aliasing
- **Performance:** ~20-30% slower than Bresenham (acceptable for 3000 lines)

**Trade-offs:**
✅ **Pros:**
- Much better SVG matching (primary goal)
- Smoother visual appearance
- More accurate preview
- Still fast enough for real-time generation

❌ **Cons:**
- Slightly slower than Bresenham
- More complex implementation
- Requires float arithmetic (but still efficient)

**Alternative Considered:**
- Gupta-Sproull algorithm: More accurate but significantly slower
- Supersampling: Too slow for 3000 lines
- Post-processing blur: Doesn't match SVG behavior

**Risk Assessment:**
- **Low risk:** Well-established algorithm (1991)
- **Easy rollback:** Can keep Bresenham as fallback
- **Testing needed:** Measure actual MAE reduction

**Next Steps (for user decision):**
1. Implement Xiaolin Wu's algorithm in experimental/
2. Generate test output with Wu vs Bresenham
3. Measure Canvas-SVG MAE for both
4. Visual comparison of smoothness
5. If MAE < 20, consider replacing Bresenham in v10

**References:**
- Wu, Xiaolin (1991). "An Efficient Antialiasing Technique". Computer Graphics, 25(4).
- Wu, Xiaolin (1992). "Fast Antialiasing". Dr. Dobb's Journal.
- Wikipedia: https://en.wikipedia.org/wiki/Xiaolin_Wu%27s_line_algorithm

**Status:** Ready for prototyping
**Priority:** HIGH (directly addresses Canvas-SVG mismatch)
**Estimated dev time:** 2-3 hours

---

