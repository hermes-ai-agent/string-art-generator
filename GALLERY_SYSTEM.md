# Auto-Update Gallery System

## Overview

Static gallery page yang otomatis update setiap ada versi baru dari:
- Manual generation
- Self-learning experiments

## Files

### Core Scripts

1. **`update_gallery.py`** - Main script untuk update gallery
   - Copy SVG/PNG ke `docs/`
   - Update `results_manifest.json`
   - Update `baseline_metrics.json` jika ada improvement
   
2. **`generate_manifest.py`** - Regenerate manifest dari semua file di `docs/`
   - Scan semua SVG files
   - Extract version numbers
   - Load metrics dari `baseline_metrics.json`
   - Generate `docs/results_manifest.json`

3. **`self_learning_v2.py`** - Self-learning dengan auto gallery update
   - Test parameter variations
   - Measure SSIM
   - Auto-call `update_gallery.py` untuk setiap run
   - Auto-increment version numbers (v12, v13, v14, ...)

### Gallery Pages

- **`docs/gallery.html`** - Main gallery page
- **`docs/index.html`** - Landing page
- **`docs/results_manifest.json`** - Data source untuk gallery

## Usage

### Manual Generation + Gallery Update

```bash
# Generate string art
./string-art-gen --input photo.jpg --output output/result.svg --pins 300 --lines 2500 --wu

# Update gallery
./update_gallery.py output/result.svg \
  --version v12 \
  --ssim 0.234 \
  --quality 8 \
  --pins 300 \
  --lines 2500 \
  --description "Manual: Wu anti-aliased with optimized params"
```

### Self-Learning (Auto Gallery Update)

```bash
# Run self-learning (automatically updates gallery)
./self_learning_v2.py
```

Output:
- Generates string art with Go binary
- Measures SSIM
- Auto-updates gallery with new version
- Marks as "failed" if no improvement

### Regenerate Manifest

```bash
# Scan all files in docs/ and regenerate manifest
./generate_manifest.py
```

Use this when:
- Manual files added to `docs/`
- Manifest corrupted
- Need to update metrics for existing versions

## Version Numbering

- **v1-v11**: Manual versions (historical)
- **v12+**: Auto-incremented by self-learning
- Version determined by: `v{12 + len(experiment_log.json)}`

## Gallery Features

### Stats Summary
- Total versions
- Best SSIM
- Success rate
- Average generation time

### Filters
- All versions
- Excellent (SSIM ≥ 0.25)
- Good (SSIM ≥ 0.20)
- Failed (quality gate rejected)

### Version Cards
Each card shows:
- Version number
- SSIM score
- Quality badge (color-coded)
- Parameters (pins, lines, alpha)
- Description
- Timestamp
- Preview image (PNG or SVG)

### Status Badges
- 🟢 **Best** - Highest SSIM
- 🔴 **Failed** - Quality gate rejected
- 🔵 **Normal** - Successful improvement

## Baseline Tracking

`baseline_metrics.json` tracks the best version:

```json
{
  "version": "v10.0",
  "ssim": 0.1959,
  "quality_score": 7,
  "generation_time": 8.2,
  "pins": 300,
  "lines": 2500,
  "improvement": "Source-over compositing, Bresenham rasterization",
  "note": "LOCKED BASELINE - V10 is objectively the best version"
}
```

When a new version beats the baseline:
- `baseline_metrics.json` updated
- Gallery shows 🟢 **Best** badge
- Notification sent (if configured)

## Integration with Self-Learning

Self-learning automatically:
1. Generates string art with Go binary (`--wu` mode)
2. Measures SSIM with `measure_ssim.py`
3. Calls `update_gallery.py` with results
4. Updates `experiment_log.json`
5. Updates baseline if improved

## File Structure

```
string-art/
├── docs/
│   ├── gallery.html          # Gallery page
│   ├── index.html            # Landing page
│   ├── results_manifest.json # Gallery data
│   ├── result_v12_*.svg      # Generated SVGs
│   ├── result_v12_*.png      # Preview PNGs
│   └── ...
├── output/                   # Temporary generation output
├── baseline_metrics.json     # Best version tracker
├── experiment_log.json       # Self-learning history
├── update_gallery.py         # Gallery updater
├── generate_manifest.py      # Manifest generator
├── self_learning_v2.py       # Self-learning script
└── string-art-gen            # Go binary
```

## Deployment

Gallery di-deploy ke **Cloudflare Pages** (private repo friendly).

### Quick Deploy

```bash
# Manual deploy
./deploy_gallery.sh

# Auto-deploy with gallery update
./update_gallery.py output/result.svg --version v12 --ssim 0.234 --deploy
```

### Setup

See [CLOUDFLARE_DEPLOY.md](CLOUDFLARE_DEPLOY.md) for complete setup instructions.

**Production URL:** https://string-art-generator.pages.dev

## Maintenance

### Add New Version Manually

1. Generate SVG
2. Run `update_gallery.py` with metrics
3. Gallery auto-updates

### Fix Manifest

```bash
# Regenerate from all files in docs/
./generate_manifest.py
```

### Clean Old Versions

```bash
# Remove old files from docs/
rm docs/result_v{12..20}_*.{svg,png}

# Regenerate manifest
./generate_manifest.py
```

## Quality Gates

Self-learning marks versions as "failed" if:
- SSIM not improved over baseline
- Generation failed
- SSIM measurement failed

Failed versions still appear in gallery with 🔴 badge.

## Future Enhancements

- [ ] Auto-deploy to GitHub Pages on new version
- [ ] Telegram notification on new best version
- [ ] Comparison view (side-by-side)
- [ ] Download all versions as ZIP
- [ ] Filter by date range
- [ ] Search by description
- [ ] Sort by SSIM/time/pins/lines
