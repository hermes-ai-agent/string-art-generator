#!/bin/bash
# Analyze improvement over baseline

SVG_FILE="$1"
BASELINE_SSIM=0.264
BASELINE_BRIGHTNESS=111

if [ ! -f "$SVG_FILE" ]; then
    echo "ERROR: File not found: $SVG_FILE"
    exit 1
fi

echo "=== Analyzing: $SVG_FILE ==="
echo ""

# Run quality validator
python3 quality_validator.py "$SVG_FILE" cat_photo.jpg

# Extract metrics
SSIM=$(python3 quality_validator.py "$SVG_FILE" cat_photo.jpg 2>&1 | grep "^SSIM:" | awk '{print $2}')
BRIGHTNESS=$(python3 quality_validator.py "$SVG_FILE" cat_photo.jpg 2>&1 | grep "^Mean brightness:" | awk '{print $3}' | cut -d'/' -f1)

echo ""
echo "=== Comparison to Baseline ==="
echo "SSIM: $SSIM (baseline: $BASELINE_SSIM)"
echo "Brightness: $BRIGHTNESS (baseline: $BASELINE_BRIGHTNESS)"

# Calculate improvement
if [ ! -z "$SSIM" ]; then
    IMPROVEMENT=$(echo "scale=4; ($SSIM - $BASELINE_SSIM) / $BASELINE_SSIM * 100" | bc)
    echo "SSIM improvement: ${IMPROVEMENT}%"
fi

