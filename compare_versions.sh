#!/bin/bash
# Compare multiple versions side by side

echo "==================================="
echo "String Art Version Comparison"
echo "==================================="
echo ""

VERSIONS=("v6.3_tuned" "test_v26")

for version in "${VERSIONS[@]}"; do
    SVG="docs/cat_${version}.svg"
    if [ ! -f "$SVG" ]; then
        SVG="docs/${version}.svg"
    fi
    
    if [ -f "$SVG" ]; then
        echo "--- $version ---"
        python3 quality_validator.py "$SVG" cat_photo.jpg 2>&1 | grep -E "(SSIM|Mean brightness|OVERALL|VERDICT)"
        echo ""
    else
        echo "--- $version ---"
        echo "File not found: $SVG"
        echo ""
    fi
done

echo "==================================="
echo "Visual Comparison"
echo "==================================="
echo ""
echo "Check these mobile previews:"
for version in "${VERSIONS[@]}"; do
    PNG="docs/cat_${version}_mobile_400px.png"
    if [ ! -f "$PNG" ]; then
        PNG="docs/${version}_mobile_400px.png"
    fi
    
    if [ -f "$PNG" ]; then
        echo "  - $PNG"
    fi
done
echo ""
