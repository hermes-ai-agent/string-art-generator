#!/usr/bin/env python3
from PIL import Image, ImageOps
import sys

img = Image.open(sys.argv[1])
inverted = ImageOps.invert(img.convert('RGB'))
inverted.save(sys.argv[2])
print(f"✓ Inverted {sys.argv[1]} → {sys.argv[2]}")
