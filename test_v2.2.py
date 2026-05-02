#!/usr/bin/env python3
"""Quick test for v2.2.0 improvements"""

from PIL import Image, ImageDraw
import numpy as np

# Create simple test image (circle)
size = 400
img = Image.new('L', (size, size), 'white')
draw = ImageDraw.Draw(img)

# Draw a simple circle
center = size // 2
radius = 100
draw.ellipse([center-radius, center-radius, center+radius, center+radius], fill='black')

# Save
img.save('/home/amin/string-art/test_simple.png')
print("✓ Created test_simple.png")
