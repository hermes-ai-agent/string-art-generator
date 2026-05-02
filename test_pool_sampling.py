#!/usr/bin/env python3
"""
Test script for pool-based sampling (v3.9.0)
Compares performance and quality between different sampling methods
"""

import sys
import time
from PIL import Image, ImageDraw
import numpy as np

# Create simple test image
def create_test_image(size=800):
    """Create a simple test image with geometric shapes"""
    img = Image.new('L', (size, size), color=255)
    draw = ImageDraw.Draw(img)
    
    # Draw a circle
    center = size // 2
    radius = size // 3
    draw.ellipse([center-radius, center-radius, center+radius, center+radius], 
                 fill=0, outline=0)
    
    # Draw some lines
    for i in range(8):
        angle = i * np.pi / 4
        x1 = int(center + radius * 0.5 * np.cos(angle))
        y1 = int(center + radius * 0.5 * np.sin(angle))
        x2 = int(center + radius * 0.8 * np.cos(angle))
        y2 = int(center + radius * 0.8 * np.sin(angle))
        draw.line([x1, y1, x2, y2], fill=128, width=3)
    
    return img

# Test different configurations
def test_pool_sampling():
    print("=" * 70)
    print("Testing Pool-Based Sampling (v3.9.0)")
    print("=" * 70)
    
    # Create test image
    print("\n1. Creating test image...")
    img = create_test_image()
    img.save('test_pool_input.png')
    print("   ✓ Saved: test_pool_input.png")
    
    # Import generator
    sys.path.insert(0, '/home/amin/string-art')
    from string_art_generator import StringArtGenerator
    
    # Test configurations
    configs = [
        {
            'name': 'Pool Sampling (Bridges 2022)',
            'params': {
                'num_pins': 100,
                'use_pool_sampling': True,
                'pool_size': 1000,
                'pool_select': 20,
                'use_knn_selection': False,
                'random_sampling': False,
                'beam_width': 1,
                'use_parallel': True,
                'use_2opt': False,
                'use_3opt': False,
            }
        },
        {
            'name': 'k-NN Selection (v3.7.0)',
            'params': {
                'num_pins': 100,
                'use_pool_sampling': False,
                'use_knn_selection': True,
                'knn_neighbors': 30,
                'random_sampling': False,
                'beam_width': 1,
                'use_parallel': True,
                'use_2opt': False,
                'use_3opt': False,
            }
        },
        {
            'name': 'Legacy Random Sampling',
            'params': {
                'num_pins': 100,
                'use_pool_sampling': False,
                'use_knn_selection': False,
                'random_sampling': True,
                'sample_size': 500,
                'beam_width': 1,
                'use_parallel': True,
                'use_2opt': False,
                'use_3opt': False,
            }
        }
    ]
    
    results = []
    
    for config in configs:
        print(f"\n2. Testing: {config['name']}")
        print("   " + "-" * 60)
        
        # Create generator
        gen = StringArtGenerator(**config['params'])
        
        # Generate
        start_time = time.time()
        result = gen.generate('test_pool_input.png', max_lines=300, output_dir='test_pool_output')
        elapsed = time.time() - start_time
        
        results.append({
            'name': config['name'],
            'time': elapsed,
            'lines': len(gen.lines),
            'result': result
        })
        
        print(f"   ✓ Completed in {elapsed:.1f}s")
        print(f"   ✓ Generated {len(gen.lines)} lines")
    
    # Summary
    print("\n" + "=" * 70)
    print("RESULTS SUMMARY")
    print("=" * 70)
    
    for r in results:
        print(f"\n{r['name']}:")
        print(f"  Time: {r['time']:.1f}s")
        print(f"  Lines: {r['lines']}")
        print(f"  Speed: {r['lines']/r['time']:.1f} lines/sec")
    
    # Find fastest
    fastest = min(results, key=lambda x: x['time'])
    print(f"\n✓ Fastest: {fastest['name']} ({fastest['time']:.1f}s)")
    
    print("\n" + "=" * 70)
    print("Test completed! Check test_pool_output/ for visual results.")
    print("=" * 70)

if __name__ == '__main__':
    test_pool_sampling()
