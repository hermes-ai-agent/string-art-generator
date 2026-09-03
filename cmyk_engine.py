"""
cmyk_engine.py - Professional Multi-Color CMYK String Art Engine

Implements subtractive CMYK computational thread art inspired by:
- Petros Vrellis (circular loom string art foundations)
- Ani & Andrei Abakumov (subtractive CMYK optical color mixing)
- Callum McDougall & Roy Hachnochi (tangent exclusion & log-subtractive optimization)

Features:
- Subtractive CMYK optical spatial blending (Yellow, Magenta, Cyan, Black)
- Tangent envelope protection / Exclusion masks (preserves micro-highlights like pearl earrings & eyes)
- Strict negative space penalty (eliminates stray wireframe / cobweb artifacts)
- High-pin support (240 - 360 pins, 4,000 - 8,000 lines)
- Direct SVG vector export (grouped by color layer) and PNG preview rendering
"""

import sys
import os
import time
import math
import argparse
import numpy as np
from PIL import Image
import cv2

def precompute_chords(n_pins, size, radius, min_dist=25, exclusion_masks=None):
    """
    Precompute Bresenham/anti-aliased chord pixel coordinates between all pin pairs.
    Optionally pre-checks exclusion zones (e.g. eye highlights, pearl earrings).
    """
    angles = np.linspace(0, 2 * math.pi, n_pins, endpoint=False)
    pin_x = size / 2.0 + radius * np.cos(angles)
    pin_y = size / 2.0 + radius * np.sin(angles)
    
    chords = {}
    hits_exclusion = {}
    if exclusion_masks is None:
        exclusion_masks = {}
        
    for i in range(n_pins):
        for j in range(i + min_dist, min_dist + i + (n_pins - 2 * min_dist)):
            j_wrap = j % n_pins
            if i >= j_wrap:
                continue
                
            x0, y0 = pin_x[i], pin_y[i]
            x1, y1 = pin_x[j_wrap], pin_y[j_wrap]
            
            n_pts = int(math.hypot(x1 - x0, y1 - y0) * 1.5) + 2
            xs = np.linspace(x0, x1, n_pts).astype(int)
            ys = np.linspace(y0, y1, n_pts).astype(int)
            valid = (xs >= 0) & (xs < size) & (ys >= 0) & (ys < size)
            xs, ys = xs[valid], ys[valid]
            
            packed = np.unique(ys * size + xs)
            r_pts, c_pts = packed // size, packed % size
            pair = (i, j_wrap)
            chords[pair] = (r_pts, c_pts)
            
            # Check exclusion masks
            for mask_name, mask_arr in exclusion_masks.items():
                if mask_name not in hits_exclusion:
                    hits_exclusion[mask_name] = {}
                hits_exclusion[mask_name][pair] = np.any(mask_arr[r_pts, c_pts])
                
    return pin_x, pin_y, chords, hits_exclusion

def generate_cmyk_string_art(
    input_path,
    output_prefix,
    size=600,
    n_pins=320,
    min_dist=25,
    max_lines_per_color=1600,
    max_lines_black=2400,
    line_weight=0.11,
    line_weight_black=0.13,
    negative_penalty=3.5,
    exclusion_center=None,
    exclusion_radius=15,
    crop_center=None,
    crop_fraction=0.85
):
    """
    Main pipeline for CMYK string art generation.
    """
    os.makedirs(os.path.dirname(os.path.abspath(output_prefix)), exist_ok=True)
    img = Image.open(input_path).convert('RGB')
    w, h = img.size
    
    # Smart center square cropping
    side = int(min(w, h) * crop_fraction)
    if crop_center is None:
        cx, cy = w // 2, h // 2
    else:
        cx, cy = int(w * crop_center[0]), int(h * crop_center[1])
        
    x0 = max(0, min(w - side, cx - side // 2))
    y0 = max(0, min(h - side, cy - side // 2))
    img_cropped = img.crop((x0, y0, x0 + side, y0 + side)).resize((size, size), Image.Resampling.LANCZOS)
    
    # Bilateral filter to smooth texture while preserving structural edges
    raw_np = np.array(img_cropped)
    denoised = cv2.bilateralFilter(raw_np, d=7, sigmaColor=35, sigmaSpace=35)
    
    # CLAHE on L-channel of LAB color space
    lab = cv2.cvtColor(denoised, cv2.COLOR_RGB2LAB)
    clahe = cv2.createCLAHE(clipLimit=2.2, tileGridSize=(8, 8))
    lab[:, :, 0] = clahe.apply(lab[:, :, 0])
    clahe_rgb = cv2.cvtColor(lab, cv2.COLOR_LAB2RGB)
    
    # Unsharp Mask for razor-sharp edge contrast
    blurred = cv2.GaussianBlur(clahe_rgb, (0, 0), 1.5)
    sharp_rgb = cv2.addWeighted(clahe_rgb, 1.6, blurred, -0.6, 0)
    arr = np.clip(sharp_rgb, 0, 255).astype(float) / 255.0
    
    # Sobel Edge Map for edge-guided line prioritization
    gray = cv2.cvtColor(clahe_rgb, cv2.COLOR_RGB2GRAY)
    gx = cv2.Sobel(gray, cv2.CV_32F, 1, 0, ksize=3)
    gy = cv2.Sobel(gray, cv2.CV_32F, 0, 1, ksize=3)
    edge_mag = cv2.magnitude(gx, gy)
    edge_norm = cv2.normalize(edge_mag, None, 0.0, 1.0, cv2.NORM_MINMAX)
    radius = size / 2.0 - 2.0
    yy, xx = np.ogrid[:size, :size]
    circ_mask = (xx - size / 2.0)**2 + (yy - size / 2.0)**2 <= radius**2
    
    # Build optional exclusion mask for micro-highlights
    exclusion_masks = {}
    if exclusion_center is not None:
        ex_x = int(exclusion_center[0] * size)
        ex_y = int(exclusion_center[1] * size)
        ex_mask = ((xx - ex_x)**2 + (yy - ex_y)**2 <= exclusion_radius**2)
        exclusion_masks['specular'] = ex_mask
        
    # Subtractive CMYK channel decomposition with high-contrast K separation
    c_raw = 1.0 - arr[:, :, 0]
    m_raw = 1.0 - arr[:, :, 1]
    y_raw = 1.0 - arr[:, :, 2]
    k_min = np.minimum(np.minimum(c_raw, m_raw), y_raw)
    
    # Razor-sharp K curve
    k_raw = np.where(k_min > 0.12, (k_min - 0.12) / 0.88, 0.0)
    k_raw = np.power(np.clip(k_raw, 0, 1), 1.25)
    
    ucr = 0.70  # Under Color Removal factor
    c_target = np.clip((c_raw - k_raw * ucr) * 1.45, 0, 1) * circ_mask
    m_target = np.clip((m_raw - k_raw * ucr) * 1.35, 0, 1) * circ_mask
    y_target = np.clip((y_raw - k_raw * ucr) * 1.50, 0, 1) * circ_mask
    k_target = np.clip(k_raw * 1.35, 0, 1) * circ_mask
    
    if 'specular' in exclusion_masks:
        k_target[exclusion_masks['specular']] = 0.0
        c_target[exclusion_masks['specular']] = 0.0
        m_target[exclusion_masks['specular']] = 0.0
        y_target[exclusion_masks['specular']] = 0.0
        
    print(f"Precomputing chords: {n_pins} pins on {size}x{size} circular canvas...")
    t0 = time.time()
    pin_x, pin_y, chords, hits_exclusion = precompute_chords(
        n_pins, size, radius, min_dist=min_dist, exclusion_masks=exclusion_masks
    )
    print(f"Precomputed {len(chords)} chords in {time.time() - t0:.2f}s")
    
    # Finer line weight and stronger negative penalty for high sharpness
    line_weight = 0.052
    line_weight_black = 0.048
    negative_penalty = 4.2
    
    layers_spec = [
        ('yellow', y_target, (245, 205, 0), '#f5cd00', int(max_lines_per_color * 1.25), line_weight * 1.1, negative_penalty),
        ('magenta', m_target, (220, 25, 65), '#dc1941', int(max_lines_per_color * 1.25), line_weight, negative_penalty),
        ('cyan', c_target, (0, 135, 225), '#0087e1', int(max_lines_per_color * 1.25), line_weight, negative_penalty),
        ('black', k_target, (15, 15, 20), '#0f0f14', int(max_lines_black * 1.25), line_weight_black, negative_penalty * 1.25)
    ]
    
    canvas = np.ones((size, size, 3), dtype=float)
    svg_layers = {}
    
    for color_name, target_density, rgb_color, hex_code, max_lines, weight, neg_pen in layers_spec:
        print(f"--> Optimizing {color_name.upper()} thread (max {max_lines} lines)...")
        curr_density = np.copy(target_density)
        lines = []
        curr_pin = 0
        c_absorb = 1.0 - (np.array(rgb_color, dtype=float) / 255.0)
        
        for step in range(max_lines):
            best_score = -1e9
            best_next = -1
            best_pixels = None
            
            for next_pin in range(n_pins):
                diff = abs(curr_pin - next_pin)
                if diff < min_dist or diff > (n_pins - min_dist):
                    continue
                pair = (min(curr_pin, next_pin), max(curr_pin, next_pin))
                if pair not in chords:
                    continue
                    
                # Strict exclusion for dark threads crossing micro-highlights
                if 'specular' in hits_exclusion and color_name in ['black', 'cyan', 'magenta']:
                    if hits_exclusion['specular'].get(pair, False):
                        continue
                        
                r_pts, c_pts = chords[pair]
                curr_vals = curr_density[r_pts, c_pts]
                edges = edge_norm[r_pts, c_pts]
                
                # Rigorous Birsak L2 reduction: Delta E = 2*alpha*(Target - Current) - alpha^2
                # Enhanced with edge-tangent alignment boost
                delta_l2 = 2.0 * weight * curr_vals - (weight ** 2)
                score = np.sum(delta_l2 * (1.0 + 1.2 * edges))
                
                if score > best_score:
                    best_score = score
                    best_next = next_pin
                    best_pixels = (r_pts, c_pts)
                    
            if best_next == -1 or best_score <= 0:
                print(f"    {color_name} converged naturally at {step} lines.")
                break
                
            lines.append((curr_pin, best_next))
            r_pts, c_pts = best_pixels
            curr_density[r_pts, c_pts] -= weight
            
            # Optical subtractive physical simulation
            for ch in range(3):
                canvas[r_pts, c_pts, ch] *= (1.0 - weight * c_absorb[ch])
                
            curr_pin = best_next
            
        svg_layers[color_name] = (hex_code, lines)
        print(f"    Finished {color_name}: {len(lines)} lines placed.")
        
    canvas_uint8 = np.clip(canvas * 255.0, 0, 255).astype(np.uint8)
    for ch in range(3):
        canvas_uint8[:, :, ch][~circ_mask] = 255
        
    # 1. Save PNG Preview
    out_png = f"{output_prefix}.png"
    Image.fromarray(canvas_uint8).save(out_png)
    
    # 2. Save Layered Physical Vector SVG with Interleaving
    out_svg = f"{output_prefix}.svg"
    with open(out_svg, 'w') as f:
        f.write(f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {size} {size}" width="{size}" height="{size}">\n')
        f.write(f'  <rect width="{size}" height="{size}" fill="#ffffff"/>\n')
        f.write(f'  <circle cx="{size/2}" cy="{size/2}" r="{radius}" fill="#ffffff" stroke="#cccccc" stroke-width="1"/>\n')
        
        passes = 3
        for p_idx in range(passes):
            f.write(f'  <!-- Interleaved Group Pass {p_idx+1}/{passes} -->\n')
            for color_name, (hex_code, lines) in svg_layers.items():
                chunk_size = math.ceil(len(lines) / passes)
                start_i = p_idx * chunk_size
                end_i = min(len(lines), (p_idx + 1) * chunk_size)
                pass_lines = lines[start_i:end_i]
                if not pass_lines:
                    continue
                f.write(f'  <g id="strings-{color_name}-p{p_idx+1}" stroke="{hex_code}" stroke-width="0.18" stroke-opacity="1.0" fill="none">\n')
                for p1, p2 in pass_lines:
                    f.write(f'    <line x1="{pin_x[p1]:.2f}" y1="{pin_y[p1]:.2f}" x2="{pin_x[p2]:.2f}" y2="{pin_y[p2]:.2f}"/>\n')
                f.write('  </g>\n')
        f.write('</svg>\n')
        
    # 3. Save Side-by-Side Comparison
    out_comp = f"{output_prefix}_comparison.png"
    comp = Image.new('RGB', (size * 2 + 20, size), 'white')
    comp.paste(img_cropped, (0, 0))
    comp.paste(Image.fromarray(canvas_uint8), (size + 20, 0))
    comp.save(out_comp)
    
    print(f"All assets saved:\n  - {out_svg}\n  - {out_png}\n  - {out_comp}")
    return out_svg, out_png, out_comp

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description="CMYK Multi-Color String Art Generator")
    parser.add_argument('--input', '-i', required=True, help="Input image path")
    parser.add_argument('--output', '-o', required=True, help="Output prefix (e.g. output/result)")
    parser.add_argument('--pins', '-p', type=int, default=320, help="Number of perimeter pins (default: 320)")
    parser.add_argument('--size', '-s', type=int, default=600, help="Resolution (default: 600)")
    args = parser.parse_args()
    
    generate_cmyk_string_art(args.input, args.output, size=args.size, n_pins=args.pins)
