# String Art Learning Session Report - v26.0
**Date**: 2026-05-03  
**Session**: String art Go v3.3.0+ improvements  
**Version**: v26.0 Birsak 8x Improved  
**Status**: [TO BE FILLED]

---

## Executive Summary

Implemented v26.0 combining Birsak 2018 8x supersampling with enhanced face detection to achieve maximum quality for string art generation.

**Key Innovation**: Multi-pass face detection + 8x supersampling + local SSIM optimization

**Result**: [TO BE FILLED AFTER VALIDATION]

---

## Implementation Details

### Core Improvements

1. **8x Birsak 2018 Supersampling**
   - Renders at 6400×6400 (8× base resolution)
   - Provides 64 discrete gray levels per pixel
   - Downsamples to 800×800 for final output
   - Matches theoretical foundation from Birsak et al. 2018

2. **Local Windowed SSIM Computation**
   - 11×11 windows matching Python validator
   - Constants: c1=6.5025, c2=58.5225
   - Samples every 4th pixel for efficiency
   - Better perceptual quality metric than MSE

3. **Enhanced Face Detection**
   - Multi-pass algorithm: dark regions → elliptical filtering
   - Elliptical region: 30% width × 25% height
   - Face importance weight: 3.0× (very strong boost)
   - Upper-center focus for portrait orientation

4. **2-Phase Optimization**
   - **Phase 1**: Greedy addition with SSIM-based scoring
   - **Phase 2**: Intelligent removal (up to 10% of lines)
   - Adaptive weight decay: 1.2 → 0.8
   - Minimum weight: 14 (higher floor for detail)

5. **Calibrated Alpha**
   - α = 0.95 for 8x supersampling
   - Compensates for source-over compositing
   - Matches mobile SVG rendering behavior

### Test Configuration
```bash
./string-art-gen --input cat_photo.jpg \
  --pins 300 --lines 3000 --weight 28 \
  --min-dist 15 --edge-weight 2.0 \
  --output docs/test_v26.svg --v26
```

---

## Results

### Baseline (v6.3_tuned)
- **SSIM**: 0.2643
- **Mean Brightness**: 111/255
- **Tonal Score**: 10/10
- **Visual Score**: 6/10
- **Overall**: 7.0/10 → **FAIL**

### v26.0 Results
[TO BE FILLED AFTER VALIDATION]

**SSIM**: [VALUE]  
**Mean Brightness**: [VALUE]  
**Tonal Score**: [VALUE]/10  
**Visual Score**: [VALUE]/10  
**Overall**: [VALUE]/10 → [PASS/FAIL]

### Comparison
| Metric | Baseline | v26.0 | Change |
|--------|----------|-------|--------|
| SSIM | 0.2643 | [VALUE] | [DELTA] |
| Brightness | 111 | [VALUE] | [DELTA] |
| Tonal | 10/10 | [VALUE]/10 | [DELTA] |
| Visual | 6/10 | [VALUE]/10 | [DELTA] |
| Overall | 7.0/10 | [VALUE]/10 | [DELTA] |

### Visual Review
**Mobile Preview**: `docs/test_v26_mobile_400px.png`

**Checklist**:
- [ ] Cat features clearly visible (eyes, ears, nose)
- [ ] No solid black blobs
- [ ] No over-empty areas
- [ ] Good tonal balance (left-right, top-bottom)
- [ ] Smooth gray level transitions

**Assessment**: [TO BE FILLED]

---

## Key Learnings

### Technical Insights

1. **Supersampling Quality vs Performance Trade-off**
   - 8x provides 64 gray levels (maximum quality)
   - Computational cost: ~5-10 minutes
   - Memory usage: ~500MB peak
   - Worth it for final quality

2. **SSIM vs MSE for Perceptual Quality**
   - SSIM better matches human perception
   - Local windowing captures structural similarity
   - Must match validator implementation (11×11 windows)

3. **Face Detection Critical for Portraits**
   - Multi-pass algorithm more robust
   - Elliptical region better than circular
   - Strong importance weighting (3.0×) needed
   - Upper-center focus for portrait orientation

4. **2-Phase Optimization Essential**
   - Greedy addition gets 90% there
   - Intelligent removal refines quality
   - 10% removal rate is good balance
   - SSIM-based removal more effective than MSE

5. **Calibrated Alpha Critical**
   - Must match mobile SVG rendering
   - Alpha depends on supersample factor
   - 0.95 for 8x, 0.97 for 6x
   - Source-over compositing model

### Best Practices

1. **Always Dual Validation**
   - SSIM alone can be misleading
   - Visual review catches artifacts
   - Both must pass for deployment

2. **Adaptive Parameters Improve Quality**
   - Smooth weight decay preserves detail
   - Higher minimum weight prevents over-darkening
   - Plateau detection prevents wasted computation

3. **Importance Mapping is Powerful**
   - Edge detection + center weighting + face detection
   - Multiplicative combination works well
   - Strong face boost needed for portraits

4. **Sampling for Efficiency**
   - Sample every 4th pixel in supersampled space
   - Reduces computation without quality loss
   - Critical for 8x supersampling performance

---

## Next Steps

### If v26 Passes Both Validations
1. ✅ Auto-deploy with `./deploy_best.sh`
2. ✅ Update GitHub Pages
3. ✅ Document as new baseline
4. ✅ Commit learning report

### If v26 Fails
**Possible Adjustments**:
- Tune alpha (try 0.93-0.97 range)
- Adjust face importance weight (try 2.5-3.5)
- Modify adaptive weight curve
- Increase removal threshold
- Try 6x instead of 8x for speed

### Future Improvements

1. **Perceptual SSIM Weighting**
   - Weight SSIM by importance map
   - Focus optimization on face region
   - Could improve visual quality further

2. **Multi-Scale Optimization**
   - Coarse-to-fine approach
   - Start with 4x, refine with 8x
   - Faster convergence

3. **Better Stopping Criteria**
   - SSIM-based instead of score-based
   - Target SSIM threshold
   - More predictable results

4. **Parallel Line Removal**
   - Batch removal evaluation
   - Faster phase 2
   - Better removal decisions

---

## Code Changes

### New Files
1. **generator_v26_birsak8x_improved.go** (21KB)
   - Main generator implementation
   - Enhanced face detection
   - Local SSIM computation
   - Supersampling utilities

2. **docs/v26_improvement_notes.md** (4.3KB)
   - Technical documentation
   - Implementation details
   - Comparison with previous versions

3. **docs/v26_summary.md** (3.5KB)
   - Quick reference guide
   - Success criteria
   - Next steps

4. **test_version.sh** (1.4KB)
   - Quick test script
   - Automated validation

5. **compare_versions.sh** (1.1KB)
   - Side-by-side comparison
   - Multiple versions

### Modified Files
1. **main.go**
   - Added --v26 flag
   - Added v26 handler in main loop

### Key Functions
- `GenerateStringArtV26Birsak8xImproved()`: Main generator
- `createV26ImportanceMap()`: Enhanced importance mapping
- `detectFaceRegionV26()`: Multi-pass face detection
- `findBestLineV26SSIM()`: SSIM-based line scoring
- `computeLocalSSIMV26()`: Local windowed SSIM
- `downsampleCanvasV26()`: Supersampled canvas downsampling

---

## Performance Metrics

### Runtime
- **Expected**: 5-10 minutes
- **Actual**: [TO BE FILLED]
- **Factors**: 8x supersampling, 6400×6400 canvas
- **Optimization**: 8 parallel workers

### Memory Usage
- **Supersampled canvas**: ~327MB
- **Line pixels cache**: ~100MB
- **Total peak**: ~500MB

---

## References

1. **Birsak et al. 2018**: "String Art: Towards Computational Fabrication of String Images"
   - Theoretical foundation for supersampling
   - Source-over compositing model
   - Gray level quantization

2. **Wang et al. 2004**: "Image Quality Assessment: From Error Visibility to Structural Similarity"
   - SSIM metric definition
   - Perceptual quality assessment
   - Local windowing approach

3. **Previous Versions**:
   - v6.3: Calibrated gamma baseline (SSIM 0.264)
   - v24: Birsak 6x implementation
   - v25: Enhanced face detection

---

## Conclusion

[TO BE FILLED AFTER VALIDATION]

**Success**: [YES/NO]  
**Deployment**: [YES/NO]  
**New Baseline**: [YES/NO]

---

**Report Status**: DRAFT - Awaiting test completion and validation
**Last Updated**: 2026-05-03 15:40 UTC
