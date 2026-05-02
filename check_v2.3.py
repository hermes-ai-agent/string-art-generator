#!/usr/bin/env python3
"""Syntax check for v2.3.0"""

import sys
sys.path.insert(0, '/home/amin/string-art')

try:
    # Try importing
    import string_art_generator
    print("✓ Import successful")
    
    # Try instantiating
    gen = string_art_generator.StringArtGenerator(
        num_pins=50,
        beam_width=3,
        use_parallel=True
    )
    print(f"✓ Instantiation successful")
    print(f"  - num_pins: {gen.num_pins}")
    print(f"  - beam_width: {gen.beam_width}")
    print(f"  - use_parallel: {gen.use_parallel}")
    print(f"  - num_workers: {gen.num_workers}")
    
    # Check methods exist
    assert hasattr(gen, 'detect_edges'), "Missing detect_edges"
    assert hasattr(gen, 'find_best_pins_beam_search'), "Missing find_best_pins_beam_search"
    assert hasattr(gen, 'evaluate_pin_candidate'), "Missing evaluate_pin_candidate"
    print("✓ All methods present")
    
    print("\n✓✓✓ All checks passed! v2.3.0 is ready.")
    
except Exception as e:
    print(f"✗ Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
