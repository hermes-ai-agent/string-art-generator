#!/usr/bin/env python3
"""
Test script for String Art Generator v2.1.0
Tests new features: opacity and random sampling
"""

from PIL import Image, ImageDraw
import subprocess
import time

# Create simple test image
print("Creating test image...")
img = Image.new('L', (400, 400), 'white')
draw = ImageDraw.Draw(img)

# Draw a simple circle
draw.ellipse([100, 100, 300, 300], fill='black', outline='black')

# Add some details
draw.ellipse([150, 150, 180, 180], fill='white')  # Left eye
draw.ellipse([220, 150, 250, 180], fill='white')  # Right eye
draw.arc([150, 200, 250, 280], 0, 180, fill='white', width=5)  # Smile

img.save('test_simple_face.png')
print("✓ Test image created: test_simple_face.png")

# Test 1: Standard mode (baseline)
print("\n" + "="*60)
print("TEST 1: Standard mode (opacity=1.0, no sampling)")
print("="*60)
start = time.time()
result1 = subprocess.run([
    'python3', 'string_art_generator.py',
    'test_simple_face.png',
    '--pins', '100',
    '--lines', '300',
    '--output', 'test_output_standard'
], capture_output=True, text=True)
elapsed1 = time.time() - start
print(result1.stdout)
print(f"Time: {elapsed1:.1f}s")

# Test 2: Transparent strings
print("\n" + "="*60)
print("TEST 2: Transparent strings (opacity=0.3)")
print("="*60)
start = time.time()
result2 = subprocess.run([
    'python3', 'string_art_generator.py',
    'test_simple_face.png',
    '--pins', '100',
    '--lines', '500',  # More lines with transparency
    '--opacity', '0.3',
    '--output', 'test_output_transparent'
], capture_output=True, text=True)
elapsed2 = time.time() - start
print(result2.stdout)
print(f"Time: {elapsed2:.1f}s")

# Test 3: Random sampling (speed test)
print("\n" + "="*60)
print("TEST 3: Random sampling (faster generation)")
print("="*60)
start = time.time()
result3 = subprocess.run([
    'python3', 'string_art_generator.py',
    'test_simple_face.png',
    '--pins', '200',  # More pins
    '--lines', '500',
    '--random-sampling',
    '--sample-size', '500',
    '--output', 'test_output_sampling'
], capture_output=True, text=True)
elapsed3 = time.time() - start
print(result3.stdout)
print(f"Time: {elapsed3:.1f}s")

# Test 4: Combined (transparency + sampling)
print("\n" + "="*60)
print("TEST 4: Combined (opacity=0.3 + random sampling)")
print("="*60)
start = time.time()
result4 = subprocess.run([
    'python3', 'string_art_generator.py',
    'test_simple_face.png',
    '--pins', '200',
    '--lines', '800',
    '--opacity', '0.3',
    '--random-sampling',
    '--sample-size', '500',
    '--output', 'test_output_combined'
], capture_output=True, text=True)
elapsed4 = time.time() - start
print(result4.stdout)
print(f"Time: {elapsed4:.1f}s")

# Summary
print("\n" + "="*60)
print("TEST SUMMARY")
print("="*60)
print(f"Test 1 (Standard):     {elapsed1:.1f}s")
print(f"Test 2 (Transparent):  {elapsed2:.1f}s")
print(f"Test 3 (Sampling):     {elapsed3:.1f}s")
print(f"Test 4 (Combined):     {elapsed4:.1f}s")
print("\nAll tests completed! Check output directories for results.")
