# String Art v26.0 - Quick Reference Card

## One-Line Summary
8x Birsak supersampling + enhanced face detection + local SSIM optimization

## Test Command
```bash
./string-art-gen --input cat_photo.jpg --pins 300 --lines 3000 --weight 28 --min-dist 15 --edge-weight 2.0 --output docs/test_v26.svg --v26
```

## Validate Command
```bash
python3 quality_validator.py docs/test_v26.svg cat_photo.jpg
```

## Deploy Command (if pass)
```bash
./deploy_best.sh docs/test_v26.svg "v26.0" "300 pins, N lines, weight 28"
```

## Success Criteria
✅ SSIM > 0.264 (baseline)  
✅ Visual: Cat features clearly visible, no artifacts

## Key Parameters
- **Supersample**: 8× (64 gray levels)
- **Alpha**: 0.95 (calibrated for 8x)
- **Face weight**: 3.0× (strong boost)
- **Weight decay**: 1.2 → 0.8
- **Min weight**: 14
- **Removal**: Up to 10%

## Expected Runtime
5-10 minutes (8x supersampling is intensive)

## Expected Memory
~500MB peak (6400×6400 canvas)

## Baseline to Beat
- SSIM: 0.2643
- Brightness: 111
- Visual: 6/10

## Target
- SSIM: > 0.27
- Brightness: ~102
- Visual: 8+/10

## Files
- Implementation: `generator_v26_birsak8x_improved.go`
- Docs: `docs/v26_improvement_notes.md`
- Report: `docs/learning_session_v26_final.md`

## Quick Comparison
```bash
./compare_versions.sh
```

## Quick Test
```bash
./test_version.sh v26 300 3000 28 15 2.0
```
