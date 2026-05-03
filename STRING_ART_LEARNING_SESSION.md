# String Art Learning Session - May 3, 2026

## Objective
Improve string art generator beyond baseline SSIM 0.264 (v6.3_tuned) by implementing v3.3.0+ improvements with proper SSIM optimization.

## Key Discoveries

### 1. Critical Bug in v22: Wrong Compositing Model
**Problem**: v22 used additive subtraction (`canvas -= darkness`) instead of multiplicative darkening.

**Evidence**:
- v22 output: brightness 246.5/255 (almost white), SSIM -0.0055 (negative!)
- Baseline v6.3: brightness 111/255, SSIM 0.264
- Root cause: v22 switched from source-over compositing to additive subtraction

**Correct Approach** (from v5 baseline):
```go
// Source-over compositing (multiplicative darkening)
lineAlpha := weight / 255.0
effectiveAlpha := lineAlpha * pixel.Weight
canvas[y][x] *= (1.0 - effectiveAlpha)
```

**Wrong Approach** (v22 bug):
```go
// Additive subtraction (doesn't match SVG rendering)
darkness := weight * pixel.Weight * alpha
canvas[y][x] -= darkness
```

### 2. Local SSIM is Too Slow for Production
**Problem**: Implementing true local windowed SSIM (11x11 windows) in v23 took 7+ minutes for pre-computation alone.

**Analysis**:
- 800x800 image = 640,000 pixels
- Each window: 11x11 = 121 pixels
- ~640,000 windows to compute
- For 3000 lines × 300 candidate evaluations = 900,000 SSIM computations
- Impractical for cron job (needs to complete in <10 minutes)

**Pragmatic Solution**: Use v5's fast MSE-based scoring with importance weighting, only compute SSIM for final validation.

### 3. Best Known Parameters (v7.0 - SSIM 0.2705)
From git history analysis:
- Generator: V5 (source-over compositing)
- Pins: 300
- Lines: 3430 (not 3000)
- Weight: 28
- Min-distance: 10 (not 15)
- Edge weight: 2.0
- Over-darkening penalty: 4.0x

**Why this works**:
- More lines (3430 vs 3000) = more detail
- Smaller min-distance (10 vs 15) = more line position options
- V5's source-over compositing matches actual SVG rendering
- Over-darkening penalty prevents black blobs

## Implementation Status

### v23 (SSIM-Optimized) - ABANDONED
**Status**: Too slow for production use
**Reason**: Local windowed SSIM computation is O(n²) expensive
**Learning**: Perceptual metrics are great for validation, but too slow for line-by-line optimization

### v5 (Optimized Parameters) - TESTING
**Status**: Running with v7.0 parameters
**Expected**: SSIM ~0.270 (matching v7.0 historical best)
**Advantage**: Fast, proven, uses correct source-over compositing

## Validation Results

### Baseline (v6.3_tuned)
```
SSIM: 0.2643
Mean brightness: 111/255
Tonal Score: 10/10
Overall: 7.0/10
Verdict: FAIL (SSIM too low)
```

### v22 (Broken)
```
SSIM: -0.0055 (negative!)
Mean brightness: 246.5/255 (almost white)
Problem: Used additive subtraction instead of multiplicative darkening
```

### v5 Optimized (In Progress)
```
Expected SSIM: ~0.270
Expected brightness: ~101/255
Expected tonal: 10/10
Expected overall: ~7.5/10
```

## Next Steps

1. **If v5 optimized passes** (SSIM >0.27):
   - Deploy to GitHub Pages
   - Document parameters as new baseline
   - Consider this the practical optimum

2. **If v5 optimized fails** (SSIM <0.27):
   - Try more lines (3600, 3800)
   - Adjust weight (26, 30)
   - Fine-tune min-distance (8, 12)

3. **Future improvements** (not for this session):
   - Implement fast SSIM approximation (downsampled, sampled windows)
   - GPU-accelerated line evaluation
   - Pre-trained importance maps from face detection models

## Key Learnings

1. **Correctness > Sophistication**: v5's simple source-over compositing beats v22's complex supersampling because v22 had a fundamental bug.

2. **Validate Early**: Should have tested v22 output before implementing v23. The negative SSIM was a red flag.

3. **Performance Matters**: Local SSIM is theoretically better but practically unusable for real-time optimization.

4. **Historical Data is Gold**: Git history showed v7.0 already achieved SSIM 0.2705 with v5 + better parameters.

5. **Match the Target**: Internal canvas rendering must match actual SVG rendering (source-over compositing).

## Files Modified

- `generator_v23_ssim.go` - Created (but too slow for production)
- `main.go` - Added --v23 flag
- `V23_IMPLEMENTATION_NOTES.md` - Documentation
- `STRING_ART_LEARNING_SESSION.md` - This file

## Time Spent

- Analysis: 30 min
- v23 implementation: 45 min
- Debugging v22 bug: 20 min
- Testing: 15 min (ongoing)
- Documentation: 10 min
- **Total**: ~2 hours

## Conclusion

The best path forward is not always the most sophisticated. v5 with optimized parameters (v7.0 config) is likely the practical optimum for this problem. The key insight is that **source-over compositing** is essential for matching SVG rendering, and v22's switch to additive subtraction was a critical regression.
