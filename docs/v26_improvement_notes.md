# String Art Generator v26.0 - Birsak 8x Improved

## Improvement Summary

Version 26.0 combines the best practices from v24 (Birsak 8x supersampling) and v25 (enhanced face detection) to achieve maximum quality.

## Key Improvements

### 1. **8x Birsak 2018 Supersampling** (from v24)
- Renders at 8x resolution (6400x6400 for 800x800 base)
- Provides 64 discrete gray levels per pixel (8x8 subpixels)
- Downsamples to base resolution for final output
- Matches the theoretical foundation from Birsak et al. 2018 paper

### 2. **Local Windowed SSIM Computation** (from v24)
- Uses 11x11 windows matching Python validator
- Computes structural similarity index (SSIM) instead of MSE
- Better perceptual quality metric
- Constants: c1 = 6.5025, c2 = 58.5225

### 3. **Enhanced Face Detection** (improved from v25)
- Multi-pass detection algorithm
- First pass: identifies dark regions (< 80% avg brightness)
- Second pass: filters by elliptical distance from face center
- Elliptical region: 30% width × 25% height
- Face importance weight: 3.0× (very strong boost)

### 4. **2-Phase Optimization** (from v24)
- **Phase 1: Greedy Addition**
  - SSIM-based line scoring
  - Adaptive weight decay: 1.2 → 0.8 (smoother than v24)
  - Minimum weight: 14 (higher floor for detail preservation)
  - Adaptive stopping on score plateau
  
- **Phase 2: Intelligent Removal**
  - Samples up to 800 lines for removal candidates
  - Removes lines that improve SSIM
  - Maximum removals: 10% of total lines
  - Improvement threshold: 0.00005 (very fine for 8x)

### 5. **Calibrated Alpha for Mobile SVG** (from v24)
- Alpha = 0.95 for 8x supersampling
- Compensates for source-over compositing
- Matches actual mobile SVG rendering behavior

## Technical Details

### Supersampling Implementation
```go
supersample := 8
superWidth := width * supersample  // 800 → 6400
superHeight := height * supersample
```

### SSIM Scoring
```go
// Sample every 4th pixel in supersampled space for efficiency
sampleStep := 4
for i := 0; i < len(pixels); i += sampleStep {
    // Downsample coordinates to base resolution
    baseX := p.X / supersample
    baseY := p.Y / supersample
    
    // Calculate error reduction with importance weighting
    errorReduction := oldError - newError
    if errorReduction > 0 {
        totalScore += errorReduction * importance[baseY][baseX]
    }
}
```

### Face Detection Algorithm
```go
// Multi-pass detection
// Pass 1: Find dark regions (< 80% avg brightness)
if brightness < avgBrightness*0.80 {
    darkRegions[y][x] = true
}

// Pass 2: Filter by elliptical distance
normalizedDist := sqrt((dx²/maxDistX²) + (dy²/maxDistY²))
if normalizedDist < 1.0 {
    faceRegion[y][x] = true
}
```

## Expected Results

### Target Metrics
- **SSIM**: > 0.27 (baseline: 0.264)
- **Mean Brightness**: ~102 (baseline: 111)
- **Visual Quality**: Clear cat features (eyes, ears, nose visible)

### Improvements Over Baseline
- Better tonal balance (left-right, top-bottom)
- Clearer facial features
- No solid black blobs
- No over-empty areas
- Smoother gray level transitions

## Build & Test

```bash
cd /home/amin/string-art
go build -o string-art-gen .

# Test v26
./string-art-gen --input cat_photo.jpg \
  --pins 300 --lines 3000 --weight 28 \
  --min-dist 15 --edge-weight 2.0 \
  --output docs/test_v26.svg --v26

# Validate quality
python3 quality_validator.py docs/test_v26.svg cat_photo.jpg
```

## Validation Criteria

Both must pass:

1. **SSIM Score**: Must be > 0.264 (baseline)
2. **Visual Review**: 
   - Check `docs/test_v26_mobile_400px.png`
   - Cat features must be clearly visible
   - No visual artifacts
   - Good tonal balance

If both pass → auto-deploy with `./deploy_best.sh`

## Comparison with Previous Versions

| Version | Approach | SSIM | Visual | Notes |
|---------|----------|------|--------|-------|
| v6.3 (baseline) | V5 + calibrated gamma | 0.264 | 6/10 | Current best |
| v24 | Birsak 6x + SSIM | ? | ? | Good foundation |
| v25 | V5 + enhanced face | ? | ? | Better face detection |
| **v26** | **Birsak 8x + enhanced face** | **?** | **?** | **Best of both** |

## References

- Birsak et al. 2018: "String Art: Towards Computational Fabrication of String Images"
- SSIM: Wang et al. 2004: "Image Quality Assessment: From Error Visibility to Structural Similarity"
