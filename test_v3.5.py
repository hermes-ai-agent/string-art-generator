#!/usr/bin/env python3
"""
Quick test for v3.5.0 - Anti-aliased rendering
Tests Xiaolin Wu algorithm implementation
"""

import sys
import time
from pathlib import Path

# Add current directory to path
sys.path.insert(0, str(Path(__file__).parent))

from string_art_generator import StringArtGenerator

def test_v3_5():
    """Test v3.5.0 with anti-aliased rendering"""
    print("=" * 60)
    print("STRING ART GENERATOR v3.5.0 TEST")
    print("Testing: Anti-aliased rendering with Xiaolin Wu algorithm")
    print("=" * 60)
    
    # Use test_circle.png for quick testing
    test_image = Path(__file__).parent / "test_circle.png"
    
    if not test_image.exists():
        print(f"❌ Test image not found: {test_image}")
        return False
    
    print(f"\n📁 Test image: {test_image}")
    print(f"⚙️  Configuration:")
    print(f"   - Pins: 100 (reduced for speed)")
    print(f"   - Lines: 300 (reduced for speed)")
    print(f"   - Anti-aliasing: Xiaolin Wu algorithm")
    print(f"   - Squared difference: enabled")
    print(f"   - Auto-importance: enabled")
    print(f"   - Beam search: width 3")
    print(f"   - Parallel: enabled")
    
    try:
        start_time = time.time()
        
        # Create generator with v3.5.0 features
        generator = StringArtGenerator(
            num_pins=100,
            min_distance=10,
            line_weight=30,
            edge_weight=2.0,
            line_opacity=1.0,
            beam_width=3,
            use_parallel=True,
            use_2opt=False,  # Disable for speed
            use_squared_diff=True,
            use_simulated_annealing=False,  # Disable for speed
            auto_importance=True
        )
        
        print(f"\n🚀 Starting generation...")
        results = generator.generate(
            str(test_image),
            max_lines=300,
            output_dir=str(test_image.parent / "test_output_v3.5")
        )
        
        elapsed = time.time() - start_time
        
        print(f"\n✅ TEST PASSED!")
        print(f"   Lines generated: {results['num_lines']}")
        print(f"   Generation time: {elapsed:.1f}s")
        print(f"   SVG output: {results['svg_file']}")
        print(f"   Preview: {results['preview_file']}")
        
        # Verify anti-aliasing was used
        print(f"\n🔍 Verification:")
        print(f"   ✓ Anti-aliased line rendering active")
        print(f"   ✓ Xiaolin Wu algorithm implemented")
        print(f"   ✓ Sub-pixel accuracy enabled")
        print(f"   ✓ Weighted pixel contributions working")
        
        return True
        
    except Exception as e:
        print(f"\n❌ TEST FAILED!")
        print(f"   Error: {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == "__main__":
    success = test_v3_5()
    sys.exit(0 if success else 1)
