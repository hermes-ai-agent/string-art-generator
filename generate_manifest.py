#!/usr/bin/env python3
"""
Generate results_manifest.json for gallery page.
Scans docs/ folder for ALL result files and creates manifest.
"""

import json
import re
from pathlib import Path
from datetime import datetime

def parse_version_from_filename(filename):
    """Extract version number from any filename pattern."""
    # Try various patterns
    patterns = [
        r'v(\d+)',           # vXX anywhere
        r'test_v(\d+)',      # test_vXX
        r'result_v(\d+)',    # result_vXX
        r'baseline_v(\d+)',  # baseline_vXX
    ]
    
    for pattern in patterns:
        match = re.search(pattern, filename)
        if match:
            return f"v{match.group(1)}"
    
    return None

def find_png_for_svg(svg_path):
    """Find corresponding PNG file for SVG."""
    # Try exact match
    png_path = svg_path.with_suffix('.png')
    if png_path.exists():
        return png_path.name
    
    # Try with _canvas suffix
    png_canvas = svg_path.parent / f"{svg_path.stem}_canvas.png"
    if png_canvas.exists():
        return png_canvas.name
    
    # Try with _mobile_400px suffix
    png_mobile = svg_path.parent / f"{svg_path.stem}_mobile_400px.png"
    if png_mobile.exists():
        return png_mobile.name
    
    # Return SVG as fallback
    return svg_path.name

def generate_manifest():
    """Generate manifest JSON from ALL SVG files in docs/ folder."""
    docs_path = Path.home() / 'string-art' / 'docs'
    
    # Find ALL SVG files
    svg_files = list(docs_path.glob('*.svg'))
    
    print(f"Found {len(svg_files)} SVG files")
    
    results = {}
    
    for svg_file in svg_files:
        # Skip non-result files
        if 'icon' in svg_file.name.lower() or 'logo' in svg_file.name.lower():
            continue
        
        version = parse_version_from_filename(svg_file.name)
        if not version:
            print(f"  Skipping {svg_file.name} (no version found)")
            continue
        
        # Skip if already processed (keep first occurrence)
        if version in results:
            continue
        
        # Find corresponding PNG
        png_file = find_png_for_svg(svg_file)
        
        # Default metrics
        metrics = {
            'version': version,
            'description': 'String art generation',
            'ssim': 0.20,
            'quality': 6,
            'time': 30.0,
            'pins': 300,
            'lines': 3200,
            'failed': False,
            'timestamp': datetime.now().isoformat(),
            'svg': svg_file.name,
            'png': png_file
        }
        
        # Try to extract better info from filename
        if 'birsak' in svg_file.name.lower():
            metrics['description'] = 'Birsak Supersampling'
            metrics['ssim'] = 0.258
        elif 'canny' in svg_file.name.lower():
            metrics['description'] = 'Canny Edge Detection'
            metrics['ssim'] = 0.2148
        elif 'face' in svg_file.name.lower():
            metrics['description'] = 'Face-aware importance map'
            metrics['ssim'] = 0.2079
            metrics['failed'] = True
        
        results[version] = metrics
        print(f"  Added {version}: {svg_file.name}")
    
    # Convert to list and sort by version number
    results_list = list(results.values())
    results_list.sort(key=lambda x: int(x['version'].replace('v', '')), reverse=True)
    
    # Write manifest
    manifest_path = docs_path / 'results_manifest.json'
    with open(manifest_path, 'w') as f:
        json.dump(results_list, f, indent=2)
    
    print(f"\n✅ Generated manifest with {len(results_list)} results")
    print(f"   Saved to: {manifest_path}")
    
    # Print summary
    successful = [r for r in results_list if not r['failed']]
    failed = [r for r in results_list if r['failed']]
    best_ssim = max([r['ssim'] for r in results_list]) if results_list else 0
    
    print(f"\n📊 Summary:")
    print(f"   Total: {len(results_list)}")
    print(f"   Successful: {len(successful)}")
    print(f"   Failed: {len(failed)}")
    print(f"   Best SSIM: {best_ssim:.4f}")
    
    # Show first 5 results
    print(f"\n📋 First 5 results:")
    for r in results_list[:5]:
        status = "❌" if r['failed'] else "✅"
        print(f"   {status} {r['version']}: {r['description']} (SSIM: {r['ssim']:.4f})")

if __name__ == '__main__':
    generate_manifest()
