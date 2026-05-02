#!/usr/bin/env python3
"""
Syntax check only - no actual generation
"""

import sys
sys.path.insert(0, '/home/amin/string-art')

print("Checking v3.3.0 syntax...")

try:
    from string_art_generator import StringArtGenerator
    print("✓ Import successful")
    
    # Test initialization with new parameters
    gen = StringArtGenerator(
        num_pins=50,
        use_2opt=True,
        two_opt_iterations=10,
        importance_map=None
    )
    print("✓ Initialization successful")
    print(f"  - use_2opt: {gen.use_2opt}")
    print(f"  - two_opt_iterations: {gen.two_opt_iterations}")
    print(f"  - importance_map: {gen.importance_map}")
    
    # Check if method exists
    if hasattr(gen, 'apply_2opt_optimization'):
        print("✓ apply_2opt_optimization method exists")
    else:
        print("✗ apply_2opt_optimization method NOT FOUND")
        sys.exit(1)
    
    print("\n✓ All syntax checks passed!")
    
except Exception as e:
    print(f"✗ ERROR: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
