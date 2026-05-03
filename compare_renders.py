#!/usr/bin/env python3
"""Compare canvas simulation with actual SVG render"""
import numpy as np
from PIL import Image
import sys

canvas_path = sys.argv[1] if len(sys.argv) > 1 else 'docs/test_v6_lowpen_canvas.png'
svg_render_path = sys.argv[2] if len(sys.argv) > 2 else 'docs/test_v6_lowpen_mobile_400px.png'
target_path = sys.argv[3] if len(sys.argv) > 3 else 'cat_photo.jpg'

# Our canvas simulation
canvas = np.array(Image.open(canvas_path).convert('L').resize((400,400), Image.LANCZOS))
# SVG render
svg_render = np.array(Image.open(svg_render_path).convert('L'))
# Target
target = np.array(Image.open(target_path).convert('L').resize((400,400), Image.LANCZOS))

print(f'Canvas mean: {canvas.mean():.1f}')
print(f'SVG render mean: {svg_render.mean():.1f}')
print(f'Target mean: {target.mean():.1f}')
print(f'Canvas-SVG diff: {svg_render.mean() - canvas.mean():.1f} (positive = SVG lighter)')
print(f'SVG-Target diff: {svg_render.mean() - target.mean():.1f} (positive = SVG lighter)')

# Compare pixel-by-pixel
diff = svg_render.astype(float) - canvas.astype(float)
print(f'\nCanvas vs SVG pixel diff:')
print(f'  Mean: {diff.mean():.1f}')
print(f'  Std: {diff.std():.1f}')
print(f'  Max: {diff.max():.1f}')
print(f'  Min: {diff.min():.1f}')

# Brightness distribution
for threshold in [50, 100, 150, 200]:
    canvas_pct = (canvas < threshold).mean() * 100
    svg_pct = (svg_render < threshold).mean() * 100
    target_pct = (target < threshold).mean() * 100
    print(f'  Pixels < {threshold}: canvas={canvas_pct:.1f}%, svg={svg_pct:.1f}%, target={target_pct:.1f}%')
