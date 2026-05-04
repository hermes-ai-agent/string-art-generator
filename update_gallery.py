#!/usr/bin/env python3
"""
Auto-update gallery with new string art versions.
Called by self-learning or manual generation.

Usage:
  ./update_gallery.py output/result.svg --version v12 --ssim 0.234 --pins 300 --lines 2500
"""

import json
import shutil
import argparse
from pathlib import Path
from datetime import datetime
import subprocess

def copy_to_docs(svg_path, version, docs_path):
    """Copy SVG and related files to docs/ folder."""
    svg_file = Path(svg_path)
    if not svg_file.exists():
        print(f"❌ SVG file not found: {svg_path}")
        return None
    
    # Target filename: result_v{N}_timestamp.svg
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    target_name = f"result_{version}_{timestamp}.svg"
    target_svg = docs_path / target_name
    
    # Copy SVG
    shutil.copy2(svg_file, target_svg)
    print(f"✅ Copied SVG: {target_svg.name}")
    
    # Copy PNG if exists (preview)
    png_file = svg_file.with_suffix('.png')
    if png_file.exists():
        target_png = target_svg.with_suffix('.png')
        shutil.copy2(png_file, target_png)
        print(f"✅ Copied PNG: {target_png.name}")
    
    # Copy comparison if exists
    comparison_file = svg_file.parent / f"{svg_file.stem}_comparison.png"
    if comparison_file.exists():
        target_comparison = docs_path / f"{target_svg.stem}_comparison.png"
        shutil.copy2(comparison_file, target_comparison)
        print(f"✅ Copied comparison: {target_comparison.name}")
    
    return target_svg

def update_manifest(docs_path, version, svg_name, metrics):
    """Update results_manifest.json with new version."""
    manifest_path = docs_path / 'results_manifest.json'
    
    # Load existing manifest
    if manifest_path.exists():
        with open(manifest_path, 'r') as f:
            manifest = json.load(f)
    else:
        manifest = []
    
    # Remove old entries for this version (replace)
    manifest = [m for m in manifest if m['version'] != version]
    
    # Add new entry
    entry = {
        'version': version,
        'description': metrics.get('description', 'Self-learning improvement'),
        'ssim': metrics.get('ssim', 0.0),
        'quality': metrics.get('quality', 5),
        'time': metrics.get('time', 0.0),
        'pins': metrics.get('pins', 300),
        'lines': metrics.get('lines', 2500),
        'failed': metrics.get('failed', False),
        'timestamp': datetime.now().isoformat(),
        'svg': svg_name,
        'png': svg_name.replace('.svg', '.png'),
        'filename': svg_name
    }
    
    manifest.append(entry)
    
    # Sort by version (newest first)
    manifest.sort(key=lambda x: int(x['version'].replace('v', '')), reverse=True)
    
    # Write back
    with open(manifest_path, 'w') as f:
        json.dump(manifest, f, indent=2)
    
    print(f"✅ Updated manifest: {len(manifest)} versions")
    return manifest

def update_baseline_if_better(metrics, baseline_path):
    """Update baseline_metrics.json if this version is better."""
    if not baseline_path.exists():
        print("⚠️  No baseline file found")
        return False
    
    with open(baseline_path, 'r') as f:
        baseline = json.load(f)
    
    current_ssim = baseline.get('ssim', 0.0)
    new_ssim = metrics.get('ssim', 0.0)
    
    if new_ssim > current_ssim:
        baseline.update({
            'version': metrics['version'],
            'ssim': new_ssim,
            'quality_score': metrics.get('quality', 5),
            'generation_time': metrics.get('time', 0.0),
            'pins': metrics.get('pins', 300),
            'lines': metrics.get('lines', 2500),
            'improvement': metrics.get('description', 'Self-learning improvement'),
            'timestamp': datetime.now().isoformat(),
            'note': f"NEW BEST - Improved from {current_ssim:.4f} to {new_ssim:.4f} (+{((new_ssim/current_ssim - 1) * 100):.1f}%)"
        })
        
        with open(baseline_path, 'w') as f:
            json.dump(baseline, f, indent=2)
        
        print(f"🎉 NEW BASELINE! SSIM: {current_ssim:.4f} → {new_ssim:.4f} (+{((new_ssim/current_ssim - 1) * 100):.1f}%)")
        return True
    else:
        print(f"📊 Not better than baseline: {new_ssim:.4f} vs {current_ssim:.4f}")
        return False

def main():
    parser = argparse.ArgumentParser(description='Update gallery with new string art version')
    parser.add_argument('svg_path', help='Path to SVG file')
    parser.add_argument('--version', required=True, help='Version string (e.g., v12)')
    parser.add_argument('--ssim', type=float, default=0.0, help='SSIM score')
    parser.add_argument('--quality', type=int, default=5, help='Quality score (1-10)')
    parser.add_argument('--time', type=float, default=0.0, help='Generation time (seconds)')
    parser.add_argument('--pins', type=int, default=300, help='Number of pins')
    parser.add_argument('--lines', type=int, default=2500, help='Number of lines')
    parser.add_argument('--description', default='Self-learning improvement', help='Description')
    parser.add_argument('--failed', action='store_true', help='Mark as failed quality gate')
    
    args = parser.parse_args()
    
    # Paths
    project_root = Path.home() / 'string-art'
    docs_path = project_root / 'docs'
    baseline_path = project_root / 'baseline_metrics.json'
    
    print(f"\n🎨 Updating gallery for {args.version}")
    print(f"   SSIM: {args.ssim:.4f}")
    print(f"   Quality: {args.quality}/10")
    print(f"   Time: {args.time:.1f}s")
    print(f"   Pins: {args.pins}, Lines: {args.lines}")
    print()
    
    # Copy files to docs/
    target_svg = copy_to_docs(args.svg_path, args.version, docs_path)
    if not target_svg:
        return 1
    
    # Prepare metrics
    metrics = {
        'version': args.version,
        'description': args.description,
        'ssim': args.ssim,
        'quality': args.quality,
        'time': args.time,
        'pins': args.pins,
        'lines': args.lines,
        'failed': args.failed
    }
    
    # Update manifest
    manifest = update_manifest(docs_path, args.version, target_svg.name, metrics)
    
    # Update baseline if better
    if not args.failed and args.ssim > 0:
        is_new_best = update_baseline_if_better(metrics, baseline_path)
        if is_new_best:
            print("\n🏆 This is the new best version!")
    
    print(f"\n✅ Gallery updated successfully!")
    print(f"   View at: file://{docs_path}/gallery.html")
    
    return 0

if __name__ == '__main__':
    exit(main())
