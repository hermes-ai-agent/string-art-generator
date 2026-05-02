#!/usr/bin/env python3
"""
Quick test script for v3.3.0 improvements
Creates a simple test image and runs the generator
"""

from PIL import Image, ImageDraw
import numpy as np

# Create simple test image (circle)
size = 400
img = Image.new('L', (size, size), 'white')
draw = ImageDraw.Draw(img)

# Draw a filled circle
center = size // 2
radius = 150
draw.ellipse([center-radius, center-radius, center+radius, center+radius], fill='black')

# Save test image
img.save('/tmp/test_circle_v3.png')
print("✓ Created test image: /tmp/test_circle_v3.png")

# Run generator
import subprocess
result = subprocess.run([
    'python3', 
    '/home/amin/string-art/string_art_generator.py',
    '/tmp/test_circle_v3.png',
    '--pins', '100',
    '--lines', '300',
    '--output', '/tmp/string_art_test_v3'
], capture_output=True, text=True)

print("\n" + "="*60)
print("GENERATOR OUTPUT:")
print("="*60)
print(result.stdout)
if result.stderr:
    print("\nERRORS:")
    print(result.stderr)

print("\n" + "="*60)
print("Exit code:", result.returncode)
