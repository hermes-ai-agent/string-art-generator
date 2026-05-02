#!/bin/bash
# Quick test v2.2.0 with small parameters

cd ~/string-art

echo "=== Testing v2.2.0 improvements ==="
echo ""
echo "Test 1: Basic run with adaptive stopping (default)"
python3 string_art_generator.py cat_photo.jpg --pins 50 --lines 200 --output test_output_v2.2

echo ""
echo "Test 2: With look-ahead optimization"
python3 string_art_generator.py cat_photo.jpg --pins 50 --lines 200 --look-ahead --output test_output_v2.2_lookahead

echo ""
echo "=== Tests complete ==="
ls -lh test_output_v2.2*/
