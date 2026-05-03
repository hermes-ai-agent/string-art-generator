# String Art Generator v23.0 - SSIM-Optimized Implementation

## Overview
v23.0 implements true local windowed SSIM optimization to match the Python validator's quality metric, addressing the gap between v22's global SSIM and the validator's local SSIM computation.

## Critical Bug Fix: Source-Over Compositing
**Major Discovery**: v22 used additive subtraction (`canvas -= darkness`), but the baseline v6.3 uses **multiplicative darkening** (source-over compositing: `canvas *= (1 - alpha)`). This is why v22 had brightness 246.5 (almost white) instead of 111.

v23 now correctly implements source-over compositing:
- Apply: `new_brightness = old_brightness * (1 - α * aa_weight)`
- Remove: `old_brightness = new_brightness / (1 - α * aa_weight)`
- This matches how SVG renders multiple opaque strokes

## Key Improvements Over v22

### 1. Local Windowed SSIM Computation
**Problem**: v22 used global SSIM (single mean/variance for entire image), while Python validator uses local 11x11 windowed SSIM.

**Solution**: Implemented `computeLocalSSIMV23()` that:
- Uses 11x11 sliding windows (matching Python validator)
- Computes local mean, variance, and covariance for each window
- Averages SSIM across all windows
- Matches the exact SSIM formula used by scipy.ndimage.uniform_filter

### 2. SSIM-Based Line Scoring
**Problem**: v22 scored lines using simple MSE improvement, which doesn't correlate well with perceptual quality.

**Solution**: `computeLineScoreV23SSIM()` now:
- Computes MSE improvement using source-over compositing (fast baseline)
- Weights by importance map (edges + face regions)
- Adds structural similarity bonus for consistent improvements
- Rewards lines that improve many pixels consistently (structural coherence)

### 3. Intelligent Line Removal
**Problem**: v22 only checked recent lines for removal (from the end).

**Solution**: Phase 2 now:
- Samples lines throughout the sequence (500 candidates)
- Evaluates SSIM impact of removing each candidate
- Sorts by SSIM improvement potential
- Removes top candidates that actually improve SSIM
- More thorough than v22's single-pass end-removal

### 4. Correct Alpha Calibration
**Problem**: v22 used alpha=0.35 with additive subtraction, resulting in almost no darkening.

**Solution**: 
- Now uses alpha=1.0 with multiplicative darkening (source-over compositing)
- `effectiveAlpha = (weight/255) * pixel.Weight * alpha`
- With weight=28, this gives ~0.11 per-line alpha, matching v5 baseline
- Target: mean brightness ~102-111 (matching baseline)

## Technical Details

### Supersampling
- 4x supersampling (16 gray levels per pixel)
- Balanced speed/quality tradeoff
- Faster than 8x, better than 2x

### Face Detection
- Enhanced importance mapping
- Eye regions: 4x boost with gradient falloff
- Nose region: 3x boost
- Ear regions: 2.5x boost
- Edge map: 4x boost for high-contrast areas

### Optimization Phases
1. **Greedy Addition**: Add lines using SSIM-based scoring with source-over compositing
2. **Intelligent Removal**: Remove lines that hurt SSIM

## Expected Results

### Baseline (v6.3_tuned)
- SSIM: 0.2643
- Tonal Score: 10/10
- Overall: 7.0/10 (FAIL - SSIM too low)
- Brightness: 111/255

### v22 (Broken - Additive Subtraction)
- SSIM: -0.0055 (negative!)
- Brightness: 246.5/255 (almost white - barely any darkening)
- Problem: Used additive subtraction instead of multiplicative darkening

### Target for v23 (Fixed)
- SSIM: >0.27 (minimum for pass)
- SSIM: >0.35 (ideal for strong pass)
- Tonal Score: ≥8/10
- Overall: ≥6.0/10 (PASS threshold)
- Brightness: 102-111/255

## Testing Protocol

1. Generate with v23:
   ```bash
   ./string-art-gen --input cat_photo.jpg --pins 300 --lines 3000 \
     --weight 28 --min-dist 15 --edge-weight 2.0 --output docs/test_v23.svg --v23
   ```

2. Validate quality:
   ```bash
   python3 quality_validator.py docs/test_v23.svg cat_photo.jpg
   ```

3. If PASS (both SSIM and visual):
   ```bash
   ./deploy_best.sh docs/test_v23.svg "v23.0" "300 pins, 3000 lines, weight 28"
   ```

## Implementation Files
- `generator_v23_ssim.go` - Main implementation with source-over compositing
- `main.go` - Added --v23 flag
- Quality validator: `quality_validator.py`

## Next Steps if v23 Fails
1. **If SSIM still low**: Tune alpha, adjust line weight schedule, increase lines
2. **If brightness too dark**: Decrease alpha or reduce weight
3. **If brightness too light**: Increase alpha or add more lines
4. **If tonal problems**: Adjust face detection regions, edge weights
5. **If local SSIM too slow**: Consider approximations or sampling
