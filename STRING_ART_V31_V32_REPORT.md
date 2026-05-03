# String Art Learning Session Report - v31 & v32
**Date:** 2026-05-03 18:35  
**Status:** ❌ BOTH FAILED - No improvement over v30 baseline

## Session Summary

Tested 2 improvement approaches:
1. **v31:** Face-aware importance map with enhanced face detection
2. **v32:** Adaptive line weight scheduling (heavy to light)

**Result:** Both approaches FAILED to improve quality over v30 baseline.

## Baseline (v30 - Canny Edge Detection)
- **SSIM:** 0.2148
- **Brightness:** 176.0
- **Time:** 29.32s
- **Lines:** 3200
- **Quality:** 7/10 ✅ BEST

## Approach 1: v31 Face-Aware Importance Map

### Implementation
- Multi-scale face detection using edge clustering
- Adaptive importance map: 5x weight for face, 8x for eyes
- V5 greedy algorithm for line addition
- SSIM-based removal disabled (too aggressive)

### Results
- **SSIM:** 0.2079 ❌ WORSE (-3.2%)
- **Brightness:** 176.3
- **Time:** 30.66s
- **Lines:** 3200
- **Quality:** 6.5/10 ❌ FAIL

### Why It Failed
1. **Face detection too broad:** Detected region covers almost entire image (radius 399 for 800x800 image)
2. **Simple heuristic insufficient:** Grid-based edge density is not accurate face detection
3. **Wrong subject:** Algorithm designed for human faces, tested on cat photo
4. **Over-weighting ineffective:** When face covers entire image, importance map just scales uniformly

## Approach 2: v32 Adaptive Line Weight Scheduling

### Implementation
- Start with heavy lines (weight 32) for main structure
- Gradually decrease to light lines (weight 22) for details
- V5 greedy algorithm with adaptive weight per iteration
- Simple heuristic removal (remove lines with negative contribution)

### Results (Tuned: 32->22)
- **SSIM:** 0.2074 ❌ WORSE (-3.4%)
- **Brightness:** 175.7
- **Time:** 26.52s ✅ FASTER
- **Lines:** 3198 (removed 2)
- **Quality:** 6.5/10 ❌ FAIL

### Why It Failed
1. **Constant weight is better:** V30 uses constant weight 27, which is optimal
2. **Heavy lines don't help:** Starting with weight 32-35 doesn't improve structure
3. **Light lines lose contrast:** Ending with weight 20-22 loses important contrast
4. **Removal pass ineffective:** Only removed 2-13 lines, not significant

## Key Findings

### What Doesn't Work
1. **Simple heuristic face detection:** Grid-based edge density is not accurate enough
2. **Adaptive weight scheduling:** Constant weight is better than heavy-to-light schedule
3. **Complex importance maps:** Simple edge-based importance is sufficient
4. **Aggressive weighting:** 5x-8x multipliers don't improve quality

### Why v30 Is Hard to Beat
1. **Canny edge detection is excellent:** Provides high-quality edge map for importance weighting
2. **Constant weight 27 is optimal:** Not too heavy, not too light
3. **Simple greedy algorithm works well:** No need for complex optimization
4. **Fast execution:** 29s is practical for iteration

### What Might Work (Future Ideas)
1. **Multi-pass optimization:** Add/remove cycles like Birsak paper (but computationally expensive)
2. **Better line removal:** Fix SSIM-based removal to be less aggressive
3. **Parameter tuning:** Fine-tune pins/lines/weight for v30
4. **Different preprocessing:** Try different edge detection or contrast enhancement

## Comparison Table

| Version | Approach | SSIM | Brightness | Time (s) | Lines | Quality | Status |
|---------|----------|------|------------|----------|-------|---------|--------|
| v30 | Canny edge detection | 0.2148 | 176.0 | 29.3 | 3200 | 7/10 | ✅ BEST |
| v31 | Face-aware importance | 0.2079 | 176.3 | 30.7 | 3200 | 6.5/10 | ❌ FAIL |
| v32 | Adaptive weight (35->20) | 0.2071 | 174.6 | 23.4 | 3187 | 6.5/10 | ❌ FAIL |
| v32 tuned | Adaptive weight (32->22) | 0.2074 | 175.7 | 26.5 | 3198 | 6.5/10 | ❌ FAIL |

## Lessons Learned

### For String Art Optimization
1. **Simple approaches often best:** Canny edge detection + constant weight beats complex schemes
2. **Constant weight is optimal:** Adaptive scheduling doesn't improve quality
3. **Edge-based importance sufficient:** Complex face detection doesn't help
4. **SSIM 0.21-0.22 is realistic limit:** String art cannot achieve much higher SSIM with current approach

### For Future Learning Sessions
1. **Test simpler variations first:** Before complex face detection, try simpler edge enhancements
2. **Validate assumptions:** Face detection should be validated before using for importance weighting
3. **Know when to stop:** If baseline is already good, incremental improvements are hard
4. **Focus on proven techniques:** Birsak paper techniques (multi-pass, supersampling) are more promising

## Recommendations

### High Priority (More Likely to Work)
1. **Parameter tuning for v30:**
   - Test pins: 290-310 (current: 300)
   - Test lines: 3100-3300 (current: 3200)
   - Test weight: 26-28 (current: 27)
   - Test edge weight: 2.8-3.2 (current: 3.0)

2. **Fix SSIM-based removal:**
   - Current implementation too aggressive (removes almost all lines)
   - Need better contribution tracking
   - Multi-pass removal with re-evaluation

3. **Better preprocessing:**
   - Multi-scale edge detection
   - Adaptive contrast enhancement
   - Denoise + sharpen pipeline

### Low Priority (Less Likely to Work)
1. **Face detection:** Requires actual ML model, not simple heuristics
2. **Adaptive weight:** Proven to not improve quality
3. **Complex importance maps:** Simple edge-based is sufficient

## Conclusion

**Both v31 and v32 FAILED** to improve quality over v30 baseline (SSIM 0.2148).

**Root causes:**
- v31: Simple heuristic face detection is not accurate enough
- v32: Constant weight is better than adaptive scheduling

**Status:** ❌ NO DEPLOYMENT - v30 remains best version (7/10 quality)

**Next steps:** Focus on parameter tuning for v30 or fix SSIM-based removal pass.

---

## Files Generated
- `generator_v31_face_importance.go` - v31 implementation (failed)
- `generator_v32_adaptive_weight.go` - v32 implementation (failed)
- `output/test_v31.svg` - v31 result (not deployed)
- `output/test_v32.svg` - v32 result (not deployed)
- `output/test_v32_tuned.svg` - v32 tuned result (not deployed)

**Deployment:** SKIPPED (no improvement over v30)

**Learning value:** ✅ HIGH - Confirmed that simple approaches (Canny + constant weight) are optimal for string art generation.
