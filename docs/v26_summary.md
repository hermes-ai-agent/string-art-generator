# String Art v26 Implementation Summary

## Quick Reference

### What Was Done
Implemented v26.0 combining best practices from v24 (Birsak 8x supersampling) and v25 (enhanced face detection).

### Key Features
1. **8x Birsak supersampling** → 64 gray levels per pixel
2. **Local windowed SSIM** → Better perceptual quality metric
3. **Enhanced face detection** → Multi-pass elliptical algorithm
4. **2-phase optimization** → Greedy add + intelligent remove
5. **Calibrated alpha** → Matches mobile SVG rendering (α=0.95)

### Test Command
```bash
./string-art-gen --input cat_photo.jpg \
  --pins 300 --lines 3000 --weight 28 \
  --min-dist 15 --edge-weight 2.0 \
  --output docs/test_v26.svg --v26
```

### Validation Command
```bash
python3 quality_validator.py docs/test_v26.svg cat_photo.jpg
```

### Success Criteria
**Both must pass**:
1. SSIM > 0.264 (baseline)
2. Visual review: Cat features clearly visible, no artifacts

### Auto-Deploy (if both pass)
```bash
./deploy_best.sh docs/test_v26.svg "v26.0" "300 pins, N lines, weight 28"
```

---

## Technical Highlights

### Supersampling Implementation
- Base: 800×800
- Super: 6400×6400 (8× each dimension)
- Gray levels: 64 (8×8 subpixels)
- Downsampling: Average over 8×8 blocks

### Face Detection Algorithm
```
Pass 1: Find dark regions (brightness < 80% avg)
Pass 2: Filter by elliptical distance
  - Horizontal: 30% width
  - Vertical: 25% height
  - Center: upper-center region
Face importance: 3.0× boost
```

### SSIM Computation
```
Window size: 11×11 (matching Python validator)
Constants: c1=6.5025, c2=58.5225
Sampling: Every 4th pixel in supersampled space
```

### Adaptive Parameters
```
Weight decay: 1.2 → 0.8 (smoother than v24)
Minimum weight: 14 (higher floor)
Stopping: 30-window average plateau detection
Removal: Up to 10% of lines
```

---

## Expected Improvements Over Baseline

### Quantitative
- SSIM: 0.264 → >0.27 (target)
- Brightness: 111 → ~102 (target)
- Tonal score: 10/10 (maintain)

### Qualitative
- Clearer facial features (eyes, ears, nose)
- Better tonal balance
- Smoother gray transitions
- No solid black blobs
- No over-empty areas

---

## Files Created/Modified

### New Files
1. `generator_v26_birsak8x_improved.go` (21KB)
   - Main generator implementation
   - Enhanced face detection
   - Local SSIM computation
   - Supersampling utilities

2. `docs/v26_improvement_notes.md` (4.3KB)
   - Technical documentation
   - Implementation details
   - Comparison with previous versions

3. `docs/learning_session_v26_draft.md` (7.5KB)
   - Learning session report
   - Analysis and insights
   - Next steps

### Modified Files
1. `main.go`
   - Added --v26 flag
   - Added v26 handler in main loop

---

## Performance Characteristics

### Runtime
- Expected: 5-10 minutes
- Factors: 8x supersampling, 6400×6400 canvas
- Optimization: 8 parallel workers

### Memory
- Supersampled canvas: ~327MB
- Line pixels cache: ~100MB
- Total: ~500MB peak

---

## Next Steps After Validation

### If PASS (both SSIM and visual)
1. Run auto-deploy script
2. Update GitHub Pages
3. Document as new baseline
4. Commit learning report

### If FAIL
**Possible adjustments**:
- Tune alpha (0.93-0.97)
- Adjust face weight (2.5-3.5)
- Modify weight decay curve
- Try 6x instead of 8x
- Increase removal threshold

---

## References

- Birsak et al. 2018: String Art paper
- Wang et al. 2004: SSIM metric
- v6.3: Current baseline (SSIM 0.264)
- v24: Birsak 6x implementation
- v25: Enhanced face detection
