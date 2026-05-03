#!/usr/bin/env python3
from PIL import Image
import numpy as np

img = np.array(Image.open('docs/test_v5_md10_3470_mobile_400px.png').convert('L'))
h, w = img.shape
center = (h//2, w//2)
radius = min(center) - 5
y, x = np.ogrid[:h, :w]
mask = ((x - center[1])**2 + (y - center[0])**2) <= radius**2
pixels = img[mask]
print(f"Exact mean: {np.mean(pixels):.4f}")
