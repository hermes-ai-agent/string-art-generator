#!/usr/bin/env python3
"""Compare V6 baseline and V7 test renders"""
from PIL import Image
import numpy as np

# Compare the two renders
v6 = np.array(Image.open('docs/cat_v6.3_tuned_mobile_400px.png').convert('L'))
v7 = np.array(Image.open('docs/test_v7c_mobile_400px.png').convert('L'))
target = np.array(Image.open('cat_photo.jpg').convert('L').resize((400, 400), Image.LANCZOS))

print(f'V6 baseline: mean={v6.mean():.1f}, std={v6.std():.1f}')
print(f'V7 test:     mean={v7.mean():.1f}, std={v7.std():.1f}')
print(f'Target:      mean={target.mean():.1f}, std={target.std():.1f}')
print(f'Diff mean V6-target: {v6.mean()-target.mean():.1f}')
print(f'Diff mean V7-target: {v7.mean()-target.mean():.1f}')
print(f'V6 darker pixels (< 128): {(v6 < 128).sum() / v6.size * 100:.1f}%')
print(f'V7 darker pixels (< 128): {(v7 < 128).sum() / v7.size * 100:.1f}%')
print(f'Target darker pixels (< 128): {(target < 128).sum() / target.size * 100:.1f}%')

# Check contrast in different regions
h, w = v6.shape
# Top half (face area)
print(f'\nTop half (face):')
print(f'  V6: mean={v6[:h//2].mean():.1f}, std={v6[:h//2].std():.1f}')
print(f'  V7: mean={v7[:h//2].mean():.1f}, std={v7[:h//2].std():.1f}')
print(f'  Target: mean={target[:h//2].mean():.1f}, std={target[:h//2].std():.1f}')

# Bottom half
print(f'Bottom half:')
print(f'  V6: mean={v6[h//2:].mean():.1f}, std={v6[h//2:].std():.1f}')
print(f'  V7: mean={v7[h//2:].mean():.1f}, std={v7[h//2:].std():.1f}')
print(f'  Target: mean={target[h//2:].mean():.1f}, std={target[h//2:].std():.1f}')

# Center region (most important for SSIM)
cy, cx = h//2, w//2
r = h//4
center_v6 = v6[cy-r:cy+r, cx-r:cx+r]
center_v7 = v7[cy-r:cy+r, cx-r:cx+r]
center_target = target[cy-r:cy+r, cx-r:cx+r]
print(f'\nCenter region:')
print(f'  V6: mean={center_v6.mean():.1f}, std={center_v6.std():.1f}')
print(f'  V7: mean={center_v7.mean():.1f}, std={center_v7.std():.1f}')
print(f'  Target: mean={center_target.mean():.1f}, std={center_target.std():.1f}')
