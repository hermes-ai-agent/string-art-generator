# String Art v24.0 - Birsak 2018 8x Supersampling

## Implementation Date: 2026-05-03 15:00

## Key Improvements Over v23 (4x):

### 1. **8x Supersampling (64 Gray Levels)**
- Previous: 4x = 16 gray levels per pixel
- New: 8x = 64 gray levels per pixel
- Benefit: Much finer tonal gradations, better detail capture
- Birsak 2018 recommendation: 8x is optimal for quality

### 2. **Perceptual SSIM-Based Line Scoring**
- Each line evaluated by its SSIM improvement (not MSE)
- Local windowed SSIM (11x11 windows) matching Python validator
- Importance-weighted scoring for facial features

### 3. **Enhanced Face Detection**
- Higher importance weights for 8x resolution
- Eye regions: 5.0x boost (up from 4.0x)
- Nose region: 4.0x boost (up from 3.0x)
- Ear regions: 3.0x boost (up from 2.5x)
- Edge map: 5.0x boost (up from 4.0x)

### 4. **Aggressive Line Removal**
- Max removals: 300 (up from 200)
- Lower threshold: 0.0001 (down from 0.0002)
- Sample size: 600 lines (up from 500)
- More thorough quality refinement

### 5. **Calibrated Alpha for 8x**
- Alpha: 0.95 (down from 1.0)
- Compensates for higher resolution darkening
- Prevents over-darkening with 64 gray levels

### 6. **Adaptive Line Weight**
- Formula: baseWeight * (1.3 - 0.5*progress)
- Minimum: 12 (down from 15)
- More aggressive weight reduction for finer detail

## Expected Results:
- SSIM: >0.27 (baseline: 0.264)
- Brightness: ~102 (baseline: 111)
- Visual quality: Clear cat features (eyes, ears, nose)
- No black blobs or empty areas

## Technical Details:
- Supersampled resolution: 2400x2400 (8x of 300x300)
- Base resolution: 300x300
- Pins: 300 (1 per degree)
- Lines: 3000 (with removal phase)
- Line weight: 28
- Min distance: 15 pins

## Validation:
```bash
python3 quality_validator.py docs/test_v24.svg cat_photo.jpg
```

Both SSIM and visual quality must pass for deployment.
