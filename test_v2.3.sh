#!/bin/bash
# Test script for v2.3.0 improvements

echo "Installing scipy if needed..."
pip3 install scipy --quiet 2>/dev/null || true

echo ""
echo "Testing String Art Generator v2.3.0"
echo "===================================="
echo ""

# Test with small parameters for quick validation
echo "Test 1: Basic test with beam search (width=3)"
python3 ~/string-art/string_art_generator.py ~/string-art/test_circle.png \
    --pins 100 \
    --lines 200 \
    --beam-width 3 \
    --output ~/string-art/test_output_v2.3

echo ""
echo "Test 2: Greedy (beam=1) for comparison"
python3 ~/string-art/string_art_generator.py ~/string-art/test_circle.png \
    --pins 100 \
    --lines 200 \
    --beam-width 1 \
    --output ~/string-art/test_output_v2.3_greedy

echo ""
echo "Test 3: With parallel disabled"
python3 ~/string-art/string_art_generator.py ~/string-art/test_circle.png \
    --pins 100 \
    --lines 200 \
    --beam-width 3 \
    --no-parallel \
    --output ~/string-art/test_output_v2.3_noparallel

echo ""
echo "All tests completed!"
echo "Check ~/string-art/test_output_v2.3* for results"
