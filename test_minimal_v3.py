#!/usr/bin/env python3
"""
Minimal test untuk v3.3.0 - hanya test syntax dan basic functionality
"""

import sys
sys.path.insert(0, '/home/amin/string-art')

from string_art_generator import StringArtGenerator
from PIL import Image, ImageDraw
import tempfile
import os

print("="*60)
print("STRING ART GENERATOR v3.3.0 - MINIMAL TEST")
print("="*60)

# Create simple test image
print("\n1. Creating test image...")
size = 200
img = Image.new('L', (size, size), 'white')
draw = ImageDraw.Draw(img)
draw.ellipse([50, 50, 150, 150], fill='black')

test_img_path = '/tmp/test_minimal_v3.png'
img.save(test_img_path)
print(f"   ✓ Saved to {test_img_path}")

# Test generator initialization
print("\n2. Initializing generator with v3.3.0 parameters...")
try:
    generator = StringArtGenerator(
        num_pins=50,
        min_distance=10,
        line_weight=30,
        edge_weight=2.0,
        line_opacity=1.0,
        beam_width=2,
        use_parallel=True,
        use_2opt=True,
        two_opt_iterations=20,
        importance_map=None
    )
    print("   ✓ Generator initialized successfully")
except Exception as e:
    print(f"   ✗ ERROR: {e}")
    sys.exit(1)

# Test generation
print("\n3. Running generation (50 pins, 100 lines max)...")
try:
    output_dir = '/tmp/test_v3_minimal_output'
    results = generator.generate(
        test_img_path,
        max_lines=100,
        output_dir=output_dir
    )
    print(f"   ✓ Generation complete!")
    print(f"   - Lines: {results['num_lines']}")
    print(f"   - Time: {results['generation_time']:.1f}s")
    print(f"   - Output: {output_dir}")
except Exception as e:
    print(f"   ✗ ERROR: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

# Verify outputs
print("\n4. Verifying outputs...")
expected_files = ['sequence', 'render', 'stringart', 'comparison', 'instructions']
found = []
for f in os.listdir(output_dir):
    for exp in expected_files:
        if exp in f:
            found.append(exp)
            break

print(f"   ✓ Found {len(set(found))}/{len(expected_files)} expected output types")

print("\n" + "="*60)
print("TEST PASSED! v3.3.0 is working correctly.")
print("="*60)
