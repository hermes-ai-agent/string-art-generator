# Update Website Script

Script untuk auto-update `docs/index.html` dengan hasil self-learning terbaru.

## Usage

```bash
python3 update_website.py <version> <quality> <description> <pins> <lines> <time> <svg> <png> <params>
```

## Parameters

1. **version** - Version number (e.g., "v10.0")
2. **quality** - Quality score (e.g., "7/10")
3. **description** - Short improvement description (e.g., "Better edge detection")
4. **pins** - Number of pins (e.g., 300)
5. **lines** - Number of lines (e.g., 3200)
6. **time** - Generation time (e.g., "8.5s")
7. **svg** - SVG filename in docs/ folder (e.g., "result_20260503_173000.svg")
8. **png** - PNG filename in docs/ folder (e.g., "result_20260503_173000.png")
9. **params** - Parameter string (e.g., "300 pins, 3200 lines, weight 27")

## Example

```bash
python3 update_website.py \
  "v10.0" \
  "7/10" \
  "Better edge detection with Canny algorithm" \
  300 \
  3200 \
  "8.5s" \
  "result_20260503_173000.svg" \
  "result_20260503_173000.png" \
  "300 pins, 3200 lines, weight 27"
```

## What It Does

1. Reads current `docs/index.html`
2. Creates a new card with latest results at the top
3. Updates subtitle with latest version info
4. Preserves all old cards below
5. Writes updated HTML back to file

## Integration with Self-Learning

Self-learning cron job akan otomatis call script ini setelah:
1. ✅ Generate baseline
2. ✅ Implement improvement
3. ✅ Test dan validate
4. ✅ Visual comparison
5. ✅ Upload SVG+PNG ke docs/
6. **→ Update website dengan script ini**
7. ✅ Commit dan push ke GitHub
8. ✅ Auto-deploy via GitHub Actions

## Output

Script akan print:
```
✅ Website updated with v10.0
   Quality: 7/10
   SVG: result_20260503_173000.svg
   PNG: result_20260503_173000.png
```
