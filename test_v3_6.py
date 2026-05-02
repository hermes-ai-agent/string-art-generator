#!/usr/bin/env python3
"""
Test script for String Art Generator v3.6.0
Tests new features: hexagonal/square pin arrangements and adaptive beam width
"""

import subprocess
import sys
from PIL import Image, ImageDraw
import os

def create_test_image():
    """Create a simple test image with geometric shapes"""
    size = 400
    img = Image.new('L', (size, size), 255)  # White background
    draw = ImageDraw.Draw(img)
    
    # Draw a simple circle in the center
    center = size // 2
    radius = 80
    draw.ellipse([center-radius, center-radius, center+radius, center+radius], fill=0, outline=0)
    
    # Draw a smaller circle (eye-like)
    small_radius = 30
    draw.ellipse([center-small_radius, center-small_radius, center+small_radius, center+small_radius], fill=255, outline=255)
    
    test_path = '/tmp/test_circle.png'
    img.save(test_path)
    print(f"✓ Created test image: {test_path}")
    return test_path

def run_test(arrangement, pins=100, lines=300, beam_width=3):
    """Run string art generator with specified arrangement"""
    test_image = create_test_image()
    
    cmd = [
        'python3', 
        os.path.expanduser('~/string-art/string_art_generator.py'),
        test_image,
        '--pins', str(pins),
        '--lines', str(lines),
        '--pin-arrangement', arrangement,
        '--beam-width', str(beam_width),
        '--output', f'/tmp/string_art_test_{arrangement}'
    ]
    
    print(f"\n{'='*60}")
    print(f"Testing {arrangement.upper()} arrangement ({pins} pins, {lines} lines)")
    print(f"{'='*60}")
    
    result = subprocess.run(cmd, capture_output=True, text=True)
    
    if result.returncode == 0:
        print(f"✓ {arrangement} test PASSED")
        print(result.stdout)
        return True
    else:
        print(f"✗ {arrangement} test FAILED")
        print("STDOUT:", result.stdout)
        print("STDERR:", result.stderr)
        return False

def main():
    print("STRING ART GENERATOR v3.6.0 - TEST SUITE")
    print("="*60)
    
    results = {}
    
    # Test 1: Circle arrangement (baseline)
    results['circle'] = run_test('circle', pins=100, lines=300, beam_width=3)
    
    # Test 2: Hexagon arrangement (NEW in v3.6.0)
    results['hexagon'] = run_test('hexagon', pins=96, lines=300, beam_width=3)  # 96 = 6*16
    
    # Test 3: Square arrangement (NEW in v3.6.0)
    results['square'] = run_test('square', pins=100, lines=300, beam_width=3)
    
    # Summary
    print("\n" + "="*60)
    print("TEST SUMMARY")
    print("="*60)
    for arrangement, passed in results.items():
        status = "✓ PASS" if passed else "✗ FAIL"
        print(f"{arrangement.upper():10s}: {status}")
    
    all_passed = all(results.values())
    print("\n" + ("="*60))
    if all_passed:
        print("ALL TESTS PASSED ✓")
        print("\nOutput locations:")
        for arrangement in results.keys():
            print(f"  - /tmp/string_art_test_{arrangement}/")
    else:
        print("SOME TESTS FAILED ✗")
        sys.exit(1)

if __name__ == '__main__':
    main()
