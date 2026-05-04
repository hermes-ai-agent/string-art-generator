#!/usr/bin/env python3
"""
MANDATORY SSIM Validation Script
Measures actual SSIM between generated output and target image.
NO CLAIMS ALLOWED - only measured values.
"""

import sys
import subprocess
from pathlib import Path
from PIL import Image
import numpy as np
from skimage.metrics import structural_similarity as ssim

def load_image_grayscale(path):
    """Load image and convert to grayscale numpy array."""
    img = Image.open(path).convert('L')
    return np.array(img)

def measure_ssim(generated_path, target_path):
    """
    Measure ACTUAL SSIM between generated output and target.
    
    Args:
        generated_path: Path to generated string art (SVG or PNG)
        target_path: Path to target/reference image
    
    Returns:
        float: SSIM value (0.0 to 1.0)
    """
    # Convert SVG to PNG if needed using cairosvg or skip SVG support
    if generated_path.endswith('.svg'):
        try:
            import cairosvg
            png_path = generated_path.replace('.svg', '_ssim_temp.png')
            cairosvg.svg2png(url=generated_path, write_to=png_path, output_width=600, output_height=600)
            generated_path = png_path
        except ImportError:
            # If cairosvg not available, try to find corresponding PNG render
            png_path = generated_path.replace('_stringart_', '_render_')
            if Path(png_path).exists():
                generated_path = png_path
            else:
                raise RuntimeError("Cannot convert SVG: cairosvg not installed and no render PNG found")
    
    # Load images
    gen_img = load_image_grayscale(generated_path)
    target_img = load_image_grayscale(target_path)
    
    # Resize target to match generated if needed
    if gen_img.shape != target_img.shape:
        target_pil = Image.fromarray(target_img)
        target_pil = target_pil.resize(gen_img.shape[::-1], Image.LANCZOS)
        target_img = np.array(target_pil)
    
    # Calculate SSIM
    ssim_value = ssim(gen_img, target_img, data_range=255)
    
    # Cleanup temp file
    if generated_path.endswith('_ssim_temp.png'):
        Path(generated_path).unlink()
    
    return ssim_value

def main():
    if len(sys.argv) != 3:
        print("Usage: measure_ssim.py <generated_output> <target_image>")
        print("Example: measure_ssim.py output/result.svg input/cat.jpg")
        sys.exit(1)
    
    generated = sys.argv[1]
    target = sys.argv[2]
    
    if not Path(generated).exists():
        print(f"Error: Generated file not found: {generated}")
        sys.exit(1)
    
    if not Path(target).exists():
        print(f"Error: Target file not found: {target}")
        sys.exit(1)
    
    try:
        ssim_value = measure_ssim(generated, target)
        print(f"SSIM: {ssim_value:.4f}")
        
        # Output machine-readable format
        print(f"MEASURED_SSIM={ssim_value:.6f}")
        
        return ssim_value
    
    except Exception as e:
        print(f"Error measuring SSIM: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
