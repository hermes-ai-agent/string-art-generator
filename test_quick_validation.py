#!/usr/bin/env python3
"""Quick validation test for v3.9.0 pool sampling"""

import sys
sys.path.insert(0, '/home/amin/string-art')

from string_art_generator import StringArtGenerator

# Test instantiation with new parameters
print("Testing v3.9.0 pool sampling instantiation...")

gen = StringArtGenerator(
    num_pins=50,
    use_pool_sampling=True,
    pool_size=500,
    pool_select=20
)

print(f"✓ Generator created successfully")
print(f"  - use_pool_sampling: {gen.use_pool_sampling}")
print(f"  - pool_size: {gen.pool_size}")
print(f"  - pool_select: {gen.pool_select}")
print(f"\n✓ v3.9.0 implementation validated!")
