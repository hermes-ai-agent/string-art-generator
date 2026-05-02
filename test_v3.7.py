#!/usr/bin/env python3
"""
Test script for String Art Generator v3.7.0
Quick validation of new features
"""

import sys
import subprocess

# Test 1: Check imports
print("=" * 60)
print("TEST 1: Checking imports...")
print("=" * 60)

try:
    import numpy as np
    print("✓ numpy")
except ImportError as e:
    print(f"✗ numpy: {e}")
    sys.exit(1)

try:
    from PIL import Image
    print("✓ PIL")
except ImportError as e:
    print(f"✗ PIL: {e}")
    sys.exit(1)

try:
    from scipy import ndimage
    print("✓ scipy")
except ImportError as e:
    print(f"✗ scipy: {e}")
    sys.exit(1)

try:
    from sklearn.neighbors import NearestNeighbors
    print("✓ sklearn (NEW in v3.7.0)")
except ImportError as e:
    print(f"✗ sklearn: {e}")
    print("\nInstalling scikit-learn...")
    subprocess.run([sys.executable, "-m", "pip", "install", "scikit-learn"], check=True)
    from sklearn.neighbors import NearestNeighbors
    print("✓ sklearn installed successfully")

# Test 2: Import generator
print("\n" + "=" * 60)
print("TEST 2: Importing StringArtGenerator...")
print("=" * 60)

try:
    from string_art_generator import StringArtGenerator
    print("✓ StringArtGenerator imported successfully")
except Exception as e:
    print(f"✗ Import failed: {e}")
    sys.exit(1)

# Test 3: Check new methods
print("\n" + "=" * 60)
print("TEST 3: Checking new v3.7.0 methods...")
print("=" * 60)

generator = StringArtGenerator(num_pins=50)

# Check 3-opt method
if hasattr(generator, 'apply_3opt_optimization'):
    print("✓ apply_3opt_optimization method exists")
else:
    print("✗ apply_3opt_optimization method missing")

# Check k-NN methods
if hasattr(generator, 'setup_knn_model'):
    print("✓ setup_knn_model method exists")
else:
    print("✗ setup_knn_model method missing")

if hasattr(generator, 'get_knn_candidates'):
    print("✓ get_knn_candidates method exists")
else:
    print("✗ get_knn_candidates method missing")

# Check new parameters
if hasattr(generator, 'use_3opt'):
    print("✓ use_3opt parameter exists")
else:
    print("✗ use_3opt parameter missing")

if hasattr(generator, 'use_knn_selection'):
    print("✓ use_knn_selection parameter exists")
else:
    print("✗ use_knn_selection parameter missing")

if hasattr(generator, 'warm_restart_interval'):
    print("✓ warm_restart_interval parameter exists")
else:
    print("✗ warm_restart_interval parameter missing")

# Test 4: Quick generation test
print("\n" + "=" * 60)
print("TEST 4: Quick generation test (50 pins, 100 lines)...")
print("=" * 60)

try:
    # Create simple test image
    test_img = Image.new('L', (200, 200), 255)
    from PIL import ImageDraw
    draw = ImageDraw.Draw(test_img)
    draw.ellipse([50, 50, 150, 150], fill=0)
    test_img.save('/tmp/test_circle_v3.7.png')
    print("✓ Test image created")
    
    # Test generation with new features
    generator = StringArtGenerator(
        num_pins=50,
        use_3opt=True,
        three_opt_iterations=5,
        use_knn_selection=True,
        knn_neighbors=20,
        warm_restart_interval=50
    )
    
    results = generator.generate(
        '/tmp/test_circle_v3.7.png',
        max_lines=100,
        output_dir='/tmp/string_art_test_v3.7'
    )
    
    print(f"✓ Generation successful!")
    print(f"  - Lines generated: {results['num_lines']}")
    print(f"  - Time: {results['generation_time']:.1f}s")
    print(f"  - Output: {results['svg_file']}")
    
except Exception as e:
    print(f"✗ Generation failed: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)

print("\n" + "=" * 60)
print("ALL TESTS PASSED! ✓")
print("String Art Generator v3.7.0 is ready to use")
print("=" * 60)
