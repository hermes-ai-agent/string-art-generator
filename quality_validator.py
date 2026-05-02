#!/usr/bin/env python3
"""
String Art Quality Validator
Renders SVG at mobile resolution and computes SSIM + visual metrics.
BOTH must pass for a version to be considered "good".
"""
import sys
import subprocess
import numpy as np
from PIL import Image
from pathlib import Path

def render_svg_to_png(svg_path, output_path, width=400):
    """Render SVG to PNG at mobile resolution using rsvg-convert or cairosvg"""
    try:
        # Try rsvg-convert first (most accurate SVG rendering)
        subprocess.run([
            'rsvg-convert', '-w', str(width), '-h', str(width),
            svg_path, '-o', output_path
        ], check=True, capture_output=True)
        return True
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass
    
    try:
        # Fallback to cairosvg
        import cairosvg
        cairosvg.svg2png(url=svg_path, write_to=output_path, 
                        output_width=width, output_height=width)
        return True
    except ImportError:
        pass
    
    try:
        # Fallback to Inkscape
        subprocess.run([
            'inkscape', svg_path, '--export-type=png',
            f'--export-filename={output_path}',
            f'--export-width={width}'
        ], check=True, capture_output=True)
        return True
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass
    
    print("ERROR: No SVG renderer available (need rsvg-convert, cairosvg, or inkscape)")
    return False

def compute_ssim(img1, img2):
    """Compute SSIM between two grayscale images"""
    from scipy.ndimage import uniform_filter
    
    C1 = (0.01 * 255) ** 2
    C2 = (0.03 * 255) ** 2
    
    img1 = img1.astype(np.float64)
    img2 = img2.astype(np.float64)
    
    mu1 = uniform_filter(img1, size=11)
    mu2 = uniform_filter(img2, size=11)
    
    mu1_sq = mu1 ** 2
    mu2_sq = mu2 ** 2
    mu1_mu2 = mu1 * mu2
    
    sigma1_sq = uniform_filter(img1 ** 2, size=11) - mu1_sq
    sigma2_sq = uniform_filter(img2 ** 2, size=11) - mu2_sq
    sigma12 = uniform_filter(img1 * img2, size=11) - mu1_mu2
    
    ssim_map = ((2 * mu1_mu2 + C1) * (2 * sigma12 + C2)) /                ((mu1_sq + mu2_sq + C1) * (sigma1_sq + sigma2_sq + C2))
    
    return float(np.mean(ssim_map))

def analyze_tonal_distribution(img_array):
    """Analyze if the image has good tonal distribution"""
    # Crop to circle
    h, w = img_array.shape
    center = (h//2, w//2)
    radius = min(center) - 5
    y, x = np.ogrid[:h, :w]
    mask = ((x - center[1])**2 + (y - center[0])**2) <= radius**2
    
    pixels = img_array[mask]
    
    # Compute statistics
    mean_val = np.mean(pixels)
    std_val = np.std(pixels)
    
    # Histogram analysis
    hist, _ = np.histogram(pixels, bins=10, range=(0, 255))
    hist_normalized = hist / hist.sum()
    
    # Check for problems
    problems = []
    
    # Too dark overall (mean < 100)
    if mean_val < 100:
        problems.append(f"TOO DARK: mean brightness {mean_val:.0f}/255")
    
    # Too light overall (mean > 200)  
    if mean_val > 200:
        problems.append(f"TOO LIGHT: mean brightness {mean_val:.0f}/255")
    
    # Low contrast (std < 40)
    if std_val < 40:
        problems.append(f"LOW CONTRAST: std {std_val:.0f}")
    
    # Check for solid black blobs (>20% pixels below 20)
    very_dark_ratio = np.sum(pixels < 20) / len(pixels)
    if very_dark_ratio > 0.20:
        problems.append(f"BLACK BLOBS: {very_dark_ratio*100:.0f}% pixels are near-black")
    
    # Check for empty areas (>40% pixels above 240)
    very_light_ratio = np.sum(pixels > 240) / len(pixels)
    if very_light_ratio > 0.40:
        problems.append(f"TOO SPARSE: {very_light_ratio*100:.0f}% pixels are near-white")
    
    # Check left-right balance
    left_half = img_array[mask.copy()]  # simplified - use quadrants
    left_mean = np.mean(img_array[:, :w//2][mask[:, :w//2]])
    right_mean = np.mean(img_array[:, w//2:][mask[:, w//2:]])
    balance = abs(left_mean - right_mean)
    if balance > 50:
        problems.append(f"UNBALANCED: left={left_mean:.0f} vs right={right_mean:.0f} (diff={balance:.0f})")
    
    return {
        'mean': mean_val,
        'std': std_val,
        'very_dark_ratio': very_dark_ratio,
        'very_light_ratio': very_light_ratio,
        'balance_diff': balance,
        'problems': problems,
        'histogram': hist_normalized
    }

def validate_quality(svg_path, target_path, output_dir=None):
    """Full quality validation pipeline"""
    svg_path = Path(svg_path)
    target_path = Path(target_path)
    
    if output_dir is None:
        output_dir = svg_path.parent
    output_dir = Path(output_dir)
    
    # Render SVG at mobile resolution
    rendered_path = output_dir / f"{svg_path.stem}_mobile_400px.png"
    print(f"Rendering SVG at 400px (mobile)...")
    if not render_svg_to_png(str(svg_path), str(rendered_path), width=400):
        return None
    
    # Load images
    rendered = np.array(Image.open(rendered_path).convert('L'))
    target = np.array(Image.open(target_path).convert('L').resize((400, 400), Image.LANCZOS))
    
    # Compute SSIM
    ssim = compute_ssim(rendered, target)
    print(f"SSIM: {ssim:.4f}")
    
    # Analyze tonal distribution
    tonal = analyze_tonal_distribution(rendered)
    print(f"Mean brightness: {tonal['mean']:.0f}/255")
    print(f"Contrast (std): {tonal['std']:.0f}")
    print(f"Near-black pixels: {tonal['very_dark_ratio']*100:.1f}%")
    print(f"Near-white pixels: {tonal['very_light_ratio']*100:.1f}%")
    print(f"L/R balance diff: {tonal['balance_diff']:.0f}")
    
    # Problems
    if tonal['problems']:
        print(f"\nPROBLEMS DETECTED:")
        for p in tonal['problems']:
            print(f"  ❌ {p}")
    else:
        print(f"\n✅ No tonal problems detected")
    
    # Overall score
    # SSIM contributes 50%, tonal quality contributes 50%
    ssim_score = min(10, ssim * 15)  # SSIM 0.67 = 10/10
    
    tonal_score = 10.0
    tonal_score -= len(tonal['problems']) * 2  # -2 per problem
    tonal_score = max(0, tonal_score)
    
    overall = (ssim_score + tonal_score) / 2
    
    print(f"\n{'='*50}")
    print(f"SSIM Score: {ssim_score:.1f}/10")
    print(f"Tonal Score: {tonal_score:.1f}/10")
    print(f"OVERALL: {overall:.1f}/10")
    print(f"{'='*50}")
    
    # Pass/Fail
    passed = overall >= 6.0 and ssim >= 0.35 and len(tonal['problems']) <= 1
    print(f"\nVERDICT: {'✅ PASS' if passed else '❌ FAIL'}")
    
    return {
        'ssim': ssim,
        'ssim_score': ssim_score,
        'tonal': tonal,
        'tonal_score': tonal_score,
        'overall': overall,
        'passed': passed,
        'rendered_path': str(rendered_path)
    }

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print("Usage: python quality_validator.py <svg_path> <target_image_path>")
        sys.exit(1)
    
    result = validate_quality(sys.argv[1], sys.argv[2])
    if result is None:
        sys.exit(1)
    if not result['passed']:
        sys.exit(2)
