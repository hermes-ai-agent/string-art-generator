#!/bin/bash
# Test script for String Art Generator v3.4.0
# Run this to validate the new improvements

echo "=== String Art Generator v3.4.0 Test ==="
echo "Testing new features:"
echo "  - Squared difference metric"
echo "  - Simulated annealing"
echo "  - Auto-importance map generation"
echo ""

# Test 1: Basic test with all new features enabled (default)
echo "Test 1: All v3.4.0 features enabled (default)"
python3 string_art_generator.py test_circle.png \
    --pins 100 \
    --lines 500 \
    --output /tmp/string_art_test_v3.4_full

# Test 2: With simulated annealing
echo ""
echo "Test 2: With simulated annealing enabled"
python3 string_art_generator.py test_circle.png \
    --pins 100 \
    --lines 500 \
    --simulated-annealing \
    --annealing-temp 50.0 \
    --annealing-cooling 0.98 \
    --output /tmp/string_art_test_v3.4_annealing

# Test 3: Comparison - disable new features (v3.3.0 mode)
echo ""
echo "Test 3: Legacy mode (v3.3.0 features only)"
python3 string_art_generator.py test_circle.png \
    --pins 100 \
    --lines 500 \
    --no-squared-diff \
    --no-auto-importance \
    --output /tmp/string_art_test_v3.3_legacy

echo ""
echo "=== Tests Complete ==="
echo "Compare outputs in:"
echo "  /tmp/string_art_test_v3.4_full/"
echo "  /tmp/string_art_test_v3.4_annealing/"
echo "  /tmp/string_art_test_v3.3_legacy/"
