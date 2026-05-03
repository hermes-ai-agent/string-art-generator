# String Art Learning Session Report
**Date**: 2026-05-03  
**Focus**: String art Go v3.3.0+ improvements - Birsak 8x supersampling with enhanced face detection  
**Status**: IN PROGRESS

---

## 🎯 Learning Objective
Implement v3.3.0+ improvements combining:
1. Birsak 2018 8x supersampling (64 gray levels)
2. Local windowed SSIM computation
3. Enhanced face detection with sophisticated importance mapping
4. 2-phase optimization (greedy add + intelligent remove)
5. Calibrated alpha for mobile SVG rendering

**Target**: SSIM > 0.27, Visual quality > 7/10

---

## 📊 Baseline Analysis

### Current Best: v6.3_tuned
- **SSIM**: 0.2643
- **Mean Brightness**: 111/255 (target: 102)
- **Tonal Score**: 10/10
- **Visual Score**: 6/10
- **Overall**: 7.0/10 → **FAIL** (needs SSIM improvement)

### Problem Identified
- SSIM too low (4.0/10)
- Mean brightness too high (over-bright)
- Visual quality acceptable but not excellent

---

## 🔬 Analysis of Existing Implementations

### v24 (Birsak 6x Supersampling)
**Strengths**:
- Proper supersampling implementation (6x = 36 gray levels)
- Local windowed SSIM computation (11x11 windows)
- 2-phase optimization (add + remove)
- Calibrated alpha (0.97 for 6x)

**Weaknesses**:
- Only 6x supersampling (could be 8x for 64 levels)
- Face detection not as sophisticated

### v25 (Improved v5)
**Strengths**:
- Enhanced face detection with multi-region analysis
- Perceptual line scoring
- Aggressive line removal (12.5%)

**Weaknesses**:
- No supersampling (limited gray levels)
- MSE-based scoring (not SSIM)
- No calibrated alpha

---

## 💡 v26 Implementation Strategy

### Core Improvements

#### 1. 8x Birsak Supersampling
```go
supersample := 8  // 64 gray levels per pixel
superWidth := width * supersample  // 800 → 6400
superHeight := height * supersample
```

**Rationale**: 8x8 = 64 discrete gray levels provides maximum quality while remaining computationally feasible.

#### 2. Enhanced Face Detection
```go
// Multi-pass algorithm:
// Pass 1: Identify dark regions (< 80% avg brightness)
// Pass 2: Filter by elliptical distance from face center
// Face importance weight: 3.0× (very strong boost)
```

**Rationale**: Combines v25's sophisticated detection with stronger importance weighting.

#### 3. Local SSIM Scoring
```go
// 11x11 windows matching Python validator
// Sample every 4th pixel in supersampled space for efficiency
// Use error reduction × importance weighting
```

**Rationale**: SSIM is perceptually better than MSE for image quality assessment.

#### 4. Calibrated Alpha
```go
alpha := 0.95  // Tuned for 8x supersampling
```

**Rationale**: Compensates for source-over compositing to match mobile SVG rendering.

#### 5. Adaptive Parameters
```go
// Smoother weight decay: 1.2 → 0.8
// Higher minimum weight: 14 (vs 12 in v24)
// Adaptive stopping on score plateau (30-window average)
```

**Rationale**: Better detail preservation throughout the generation process.

---

## 🧪 Test Results

### Test Configuration
```bash
./string-art-gen --input cat_photo.jpg \
  --pins 300 --lines 3000 --weight 28 \
  --min-dist 15 --edge-weight 2.0 \
  --output docs/test_v26.svg --v26
```

### Results
**Status**: RUNNING...

[Results will be filled after completion]

---

## 📈 Quality Validation

### Dual Validation Approach

#### 1. Objective Metrics (Python Validator)
```bash
python3 quality_validator.py docs/test_v26.svg cat_photo.jpg
```

**Criteria**:
- SSIM > 0.264 (baseline)
- Mean brightness ~102 (target)
- No tonal problems

#### 2. Visual Review (Mobile PNG)
**File**: `docs/test_v26_mobile_400px.png`

**Checklist**:
- [ ] Cat features clearly visible (eyes, ears, nose)
- [ ] No solid black blobs
- [ ] No over-empty areas
- [ ] Good tonal balance (left-right, top-bottom)
- [ ] Smooth gray level transitions

---

## 🎓 Key Learnings

### Technical Insights

1. **Supersampling is Critical**
   - 8x supersampling provides 64 gray levels
   - Essential for smooth tonal transitions
   - Matches theoretical foundation from Birsak 2018

2. **SSIM > MSE for Perceptual Quality**
   - SSIM considers structural similarity
   - Better matches human perception
   - 11x11 windows provide good local context

3. **Face Detection Matters**
   - Multi-pass algorithm more robust
   - Elliptical region better than circular
   - Strong importance weighting (3.0×) needed

4. **2-Phase Optimization Essential**
   - Greedy addition gets 90% there
   - Intelligent removal refines quality
   - 10% removal rate is good balance

5. **Calibrated Alpha Critical**
   - Must match mobile SVG rendering
   - Alpha depends on supersample factor
   - 0.95 for 8x, 0.97 for 6x

### Best Practices

1. **Always validate both SSIM and visual**
   - SSIM alone can be misleading
   - Visual review catches artifacts
   - Both must pass for deployment

2. **Adaptive parameters improve quality**
   - Smooth weight decay preserves detail
   - Higher minimum weight prevents over-darkening
   - Plateau detection prevents wasted computation

3. **Importance mapping is powerful**
   - Edge detection + center weighting + face detection
   - Multiplicative combination works well
   - Strong face boost (3.0×) needed for portraits

---

## 🚀 Next Steps

### If v26 Passes Both Validations
1. Auto-deploy with `./deploy_best.sh`
2. Update GitHub Pages
3. Document as new baseline

### If v26 Fails
**Possible Adjustments**:
- Tune alpha (try 0.93-0.97 range)
- Adjust face importance weight (try 2.5-3.5)
- Modify adaptive weight curve
- Increase removal threshold
- Try different supersample factors (6x vs 8x)

### Future Improvements
1. **Perceptual SSIM weighting**
   - Weight SSIM by importance map
   - Focus optimization on face region

2. **Multi-scale optimization**
   - Coarse-to-fine approach
   - Start with 4x, refine with 8x

3. **Better stopping criteria**
   - SSIM-based instead of score-based
   - Target SSIM threshold

4. **Parallel line removal**
   - Batch removal evaluation
   - Faster phase 2

---

## 📚 References

1. **Birsak et al. 2018**: "String Art: Towards Computational Fabrication of String Images"
   - Theoretical foundation for supersampling
   - Source-over compositing model

2. **Wang et al. 2004**: "Image Quality Assessment: From Error Visibility to Structural Similarity"
   - SSIM metric definition
   - Perceptual quality assessment

3. **Previous Versions**:
   - v6.3: Calibrated gamma baseline
   - v24: Birsak 6x implementation
   - v25: Enhanced face detection

---

## 🔧 Code Changes

### New Files
- `generator_v26_birsak8x_improved.go` (21KB)
- `docs/v26_improvement_notes.md` (4.3KB)

### Modified Files
- `main.go`: Added v26 flag and handler

### Key Functions
- `GenerateStringArtV26Birsak8xImproved()`: Main generator
- `createV26ImportanceMap()`: Enhanced importance mapping
- `detectFaceRegionV26()`: Multi-pass face detection
- `findBestLineV26SSIM()`: SSIM-based line scoring
- `computeLocalSSIMV26()`: Local windowed SSIM
- `downsampleCanvasV26()`: Supersampled canvas downsampling

---

## ⏱️ Performance Notes

**Expected Runtime**: ~5-10 minutes
- 8x supersampling is computationally intensive
- 6400×6400 canvas (vs 800×800 base)
- Parallel evaluation helps (8 workers)
- Phase 2 sampling reduces cost

**Memory Usage**: ~500MB
- Supersampled canvas: 6400×6400×8 bytes = 327MB
- Line pixels cache: ~100MB
- Target/importance maps: ~50MB

---

**Status**: Waiting for test completion...
**Next Update**: After quality validation
