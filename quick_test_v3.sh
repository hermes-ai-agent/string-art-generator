#!/bin/bash
# Quick test v3.3.0 dengan parameter minimal

cd ~/string-art

echo "Testing String Art Generator v3.3.0..."
echo "========================================"
echo ""

python3 string_art_generator.py test_circle.png \
    --pins 80 \
    --lines 200 \
    --output /tmp/test_v3_output \
    --beam-width 2 \
    --2opt-iterations 50

echo ""
echo "========================================"
echo "Test complete! Check /tmp/test_v3_output/"
