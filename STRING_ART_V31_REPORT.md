# String Art v31.0 Learning Session Report
**Date:** 2026-05-03 18:31  
**Status:** ❌ FAILED - Face-aware importance map does not improve quality

## Objective
Implement face-aware importance map with enhanced face detection and adaptive weighting to improve string art quality.

## Approach
**v31.0: Face-Aware Importance Map**
- Multi-scale face detection using edge clustering
- Adaptive importance map: face regions get 5x weight, eyes get 8x weight
- Use proven v5 greedy algorithm for line addition
- SSIM-based removal disabled (too aggressive)

## Results

### Baseline (v30 - Canny Edge Detection)
- **SSIM:** 0.2148
- **Brightness:** 176.0
- **Time:** 29.32s
- **Lines:** 3200
- **Quality:** 7/10

### v31 (Face-Aware Importance Map)
- **SSIM:** 0.2079 ❌ WORSE (-3.2%)
- **Brightness:** 176.3 ✓ Similar
- **Time:** 30.66s ✓ Similar
- **Lines:** 3200
- **Quality:** 6.5/10 ❌ FAIL

## Analysis

### Why v31 Failed
1. **Face detection too broad:** Detected face region covers almost entire image (center 399,399, radius 399 for 800x800 image)
2. **Over-weighting:** 5x weight for face, 8x for eyes may be too aggressive
3. **Not suitable for cat photo:** Face detection algorithm designed for human faces, not cat faces
4. **Edge clustering approach:** Simple grid-based edge density doesn't accurately detect face boundaries

### Technical Issues
1. **Face detection algorithm:**
   - Uses 3x3 grid and finds cell with highest edge density
   - Assumes face is in upper half of image
   - No actual face detection (no Haar cascades, no ML model)
   - Just heuristic based on edge density

2. **Eye detection:**
   - Searches for dark regions with high edge density
   - Found 2 eye regions, but accuracy unknown
   - No validation of eye positions

3. **Importance map:**
   - When face region covers entire image, all pixels get 5x weight
   - This effectively just scales the entire importance map
   - No actual focus on important regions

## Lessons Learned

### What Didn't Work
1. **Simple heuristic face detection:** Grid-based edge density is not accurate enough
2. **Aggressive weighting:** 5x and 8x multipliers may be too high
3. **No validation:** Face detection has no accuracy validation
4. **Wrong subject:** Algorithm designed for human faces, tested on cat photo

### What Could Work Better
1. **Use actual face detection library:** OpenCV Haar cascades or ML-based face detection
2. **Validate detection:** Check if detected face region makes sense (size, position, aspect ratio)
3. **Adaptive weighting:** Start with lower multipliers (2x for face, 3x for eyes)
4. **Subject-aware:** Different detection for human faces vs cat faces vs other subjects
5. **Multi-scale edge detection:** Use multiple scales to detect face boundaries more accurately

## Recommendations for Future Improvements

### High Priority (More Likely to Work)
1. **Better edge detection preprocessing:**
   - Multi-scale edge detection
   - Adaptive thresholding
   - Edge enhancement in important regions

2. **Adaptive line weight scheduling:**
   - Start with heavy lines (weight 30-35)
   - Gradually decrease to light lines (weight 20-25)
   - Better contrast and detail

3. **Better line removal pass:**
   - Fix SSIM-based removal (currently too aggressive)
   - Use contribution tracking
   - Multi-pass removal with re-evaluation

### Low Priority (Less Likely to Work)
1. **Face detection:** Requires accurate detection library, not simple heuristics
2. **Complex importance maps:** Simple edge-based importance is sufficient
3. **Perceptual scoring:** MSE-based scoring with edge weighting works well

## Conclusion

**v31 face-aware importance map FAILED** with SSIM 0.2079 vs baseline 0.2148 (-3.2%).

**Root cause:** Simple heuristic face detection is not accurate enough and detected face region covers almost entire image, making importance map ineffective.

**Status:** ❌ NO DEPLOYMENT - v30 remains best version (7/10 quality)

**Next steps:** Try adaptive line weight scheduling or better line removal pass instead of face detection.

---

## Metrics Comparison

| Version | SSIM | Brightness | Time (s) | Lines | Quality | Status |
|---------|------|------------|----------|-------|---------|--------|
| v30 (baseline) | 0.2148 | 176.0 | 29.3 | 3200 | 7/10 | ✅ PASS |
| v31 (face-aware) | 0.2079 | 176.3 | 30.7 | 3200 | 6.5/10 | ❌ FAIL |

**Improvement:** -3.2% SSIM (worse)

---

## Files Generated
- `output/test_v31.svg` - v31 result (not deployed)
- `output/test_v31_canvas.png` - v31 canvas render (not deployed)
- `output/baseline_v30.svg` - v30 baseline for comparison
- `output/baseline_v30_canvas.png` - v30 canvas render

**Deployment:** SKIPPED (v31 did not improve quality)
