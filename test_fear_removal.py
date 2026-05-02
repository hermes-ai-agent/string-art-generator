#!/usr/bin/env python3
"""
Test script for fear removal feature (v3.8.0)
Creates a simple test image and compares results with/without fear removal
"""

from PIL import Image, ImageDraw
import numpy as np
from string_art_generator import StringArtGenerator
import time

# Create a simple test image with fine details
print("Creating test image with fine details...")
size = 400
img = Image.new('L', (size, size), color=255)
draw = ImageDraw.Draw(img)

# Draw a circle with fine radial lines (challenging for string art)
center = size // 2
radius = 150

# Main circle
draw.ellipse([center-radius, center-radius, center+radius, center+radius], 
             outline=0, width=3)

# Fine radial lines (details that fear removal should capture better)
for angle in range(0, 360, 15):
    angle_rad = np.radians(angle)
    x1 = center + int(radius * 0.7 * np.cos(angle_rad))
    y1 = center + int(radius * 0.7 * np.sin(angle_rad))
    x2 = center + int(radius * 0.9 * np.cos(angle_rad))
    y2 = center + int(radius * 0.9 * np.sin(angle_rad))
    draw.line([x1, y1, x2, y2], fill=0, width=2)

# Add some text for detail
draw.text((center-50, center-10), "TEST", fill=0)

# Save test image
test_img_path = "~/string-art/test_detail_image.png"
img.save(test_img_path.replace("~", "/home/amin"))
print(f"✓ Test image saved: {test_img_path}")

# Test 1: WITHOUT fear removal (baseline)
print("\n" + "="*60)
print("TEST 1: WITHOUT Fear Removal (Baseline)")
print("="*60)
gen1 = StringArtGenerator(
    num_pins=100,
    min_distance=10,
    line_weight=25,
    edge_weight=2.0,
    beam_width=3,
    use_parallel=True,
    use_2opt=False,
    use_3opt=False,
    use_knn_selection=True,
    knn_neighbors=30,
    use_fear_removal=False,  # DISABLED
    use_squared_diff=True,
    adaptive_stop=True,
    stop_threshold=0.3
)

start1 = time.time()
result1 = gen1.generate(test_img_path.replace("~", "/home/amin"), 
                        max_lines=300, 
                        output_dir="~/string-art/test_output_no_fear")
time1 = time.time() - start1

print(f"\n✓ Baseline completed:")
print(f"  - Lines: {len(gen1.lines)}")
print(f"  - Time: {time1:.1f}s")

# Test 2: WITH fear removal
print("\n" + "="*60)
print("TEST 2: WITH Fear Removal (v3.8.0)")
print("="*60)
gen2 = StringArtGenerator(
    num_pins=100,
    min_distance=10,
    line_weight=25,
    edge_weight=2.0,
    beam_width=3,
    use_parallel=True,
    use_2opt=False,
    use_3opt=False,
    use_knn_selection=True,
    knn_neighbors=30,
    use_fear_removal=True,  # ENABLED
    use_squared_diff=True,
    adaptive_stop=True,
    stop_threshold=0.3
)

start2 = time.time()
result2 = gen2.generate(test_img_path.replace("~", "/home/amin"), 
                        max_lines=300, 
                        output_dir="~/string-art/test_output_with_fear")
time2 = time.time() - start2

print(f"\n✓ Fear removal completed:")
print(f"  - Lines: {len(gen2.lines)}")
print(f"  - Time: {time2:.1f}s")

# Summary
print("\n" + "="*60)
print("COMPARISON SUMMARY")
print("="*60)
print(f"Baseline (no fear removal):")
print(f"  - Lines: {len(gen1.lines)}")
print(f"  - Time: {time1:.1f}s")
print(f"\nFear Removal (v3.8.0):")
print(f"  - Lines: {len(gen2.lines)}")
print(f"  - Time: {time2:.1f}s")
print(f"  - Difference: {len(gen2.lines) - len(gen1.lines):+d} lines")
print(f"\nExpected: Fear removal should capture more fine details")
print(f"Check output images in:")
print(f"  - ~/string-art/test_output_no_fear/")
print(f"  - ~/string-art/test_output_with_fear/")
