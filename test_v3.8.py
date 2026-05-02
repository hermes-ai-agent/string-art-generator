#!/usr/bin/env python3
"""
Quick test for fear removal feature (v3.8.0)
Uses existing test_circle.png for faster testing
"""

from string_art_generator import StringArtGenerator
import time
import os

test_img_path = os.path.expanduser("~/string-art/test_circle.png")

print("="*60)
print("TESTING FEAR REMOVAL FEATURE (v3.8.0)")
print("="*60)
print(f"Test image: {test_img_path}")
print()

# Test WITH fear removal
print("Running with FEAR REMOVAL enabled...")
print("-" * 60)
gen = StringArtGenerator(
    num_pins=100,
    min_distance=15,
    line_weight=25,
    edge_weight=2.0,
    beam_width=3,
    use_parallel=True,
    use_2opt=False,
    use_3opt=False,
    use_knn_selection=True,
    knn_neighbors=30,
    use_fear_removal=True,  # ENABLED - v3.8.0 feature
    use_squared_diff=True,
    adaptive_stop=True,
    stop_threshold=0.5
)

start = time.time()
result = gen.generate(test_img_path, 
                     max_lines=500, 
                     output_dir=os.path.expanduser("~/string-art/test_output_v3.8"))
elapsed = time.time() - start

print()
print("="*60)
print("TEST COMPLETED")
print("="*60)
print(f"✓ Lines generated: {len(gen.lines)}")
print(f"✓ Time: {elapsed:.1f}s")
print(f"✓ Output: ~/string-art/test_output_v3.8/")
print()
print("Fear removal feature allows algorithm to capture fine details")
print("without fear of temporarily worsening some pixels.")
