#!/usr/bin/env python3
"""
Update docs/index.html with latest self-learning results.
Called by self-learning cron after successful improvement.
"""

import sys
import os
from datetime import datetime
from pathlib import Path

def update_website(version: str, quality_score: str, improvement_desc: str, 
                   pins: int, lines: int, gen_time: str, svg_filename: str,
                   png_filename: str, params: str):
    """
    Update docs/index.html with new version info.
    
    Args:
        version: Version number (e.g., "v10.0")
        quality_score: Quality rating (e.g., "7/10")
        improvement_desc: Short description of improvement
        pins: Number of pins
        lines: Number of lines
        gen_time: Generation time (e.g., "8.5s")
        svg_filename: SVG filename in docs/ folder
        png_filename: PNG filename in docs/ folder
        params: Parameter string (e.g., "300 pins, 3200 lines, weight 27")
    """
    
    html_path = Path("~/string-art/docs/index.html").expanduser()
    
    if not html_path.exists():
        print(f"ERROR: {html_path} not found")
        sys.exit(1)
    
    # Read current HTML
    with open(html_path, 'r', encoding='utf-8') as f:
        html = f.read()
    
    # Extract quality number for badge color
    quality_num = int(quality_score.split('/')[0])
    quality_emoji = "⭐" if quality_num >= 7 else "🎯" if quality_num >= 5 else "🔧"
    
    # Create new card HTML
    new_card = f'''        <div class="card">
            <span class="version-badge new">{version} - LATEST {quality_emoji}</span>
            <span class="quality-badge">{quality_score} Quality</span>
            
            <h2>{version}: {improvement_desc}</h2>
            <p>Latest self-learning improvement - Generated on {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
            
            <div class="stats">
                <div class="stat">
                    <div class="stat-value">{pins}</div>
                    <div class="stat-label">Pins</div>
                </div>
                <div class="stat">
                    <div class="stat-value">{lines}</div>
                    <div class="stat-label">Lines</div>
                </div>
                <div class="stat">
                    <div class="stat-value">{gen_time}</div>
                    <div class="stat-label">Generation Time</div>
                </div>
                <div class="stat">
                    <div class="stat-value">{quality_score}</div>
                    <div class="stat-label">Visual Quality</div>
                </div>
            </div>
            
            <div class="image-container">
                <img src="{svg_filename}" alt="String Art {version}">
                <div class="image-label">SVG Output {version} (Zoomable)</div>
                <p style="color: #6b7280; margin-top: 10px;">{params}</p>
            </div>
            
            <div class="image-container">
                <img src="{png_filename}" alt="String Art {version} PNG Preview">
                <div class="image-label">PNG Preview {version}</div>
            </div>
        </div>
'''
    
    # Find insertion point (after subtitle, before first card)
    insert_marker = '        <div class="card">'
    insert_pos = html.find(insert_marker)
    
    if insert_pos == -1:
        print("ERROR: Could not find insertion point in HTML")
        sys.exit(1)
    
    # Insert new card at the top
    updated_html = html[:insert_pos] + new_card + "\n" + html[insert_pos:]
    
    # Update title version badge
    updated_html = updated_html.replace(
        '<p class="subtitle">High-performance Go implementation with error-reduction optimization</p>',
        f'<p class="subtitle">High-performance Go implementation - Latest: {version} ({quality_score})</p>'
    )
    
    # Write updated HTML
    with open(html_path, 'w', encoding='utf-8') as f:
        f.write(updated_html)
    
    print(f"✅ Website updated with {version}")
    print(f"   Quality: {quality_score}")
    print(f"   SVG: {svg_filename}")
    print(f"   PNG: {png_filename}")

if __name__ == "__main__":
    if len(sys.argv) != 10:
        print("Usage: update_website.py <version> <quality> <description> <pins> <lines> <time> <svg> <png> <params>")
        print("Example: update_website.py v10.0 '7/10' 'Better edge detection' 300 3200 8.5s result.svg result.png '300 pins, 3200 lines, weight 27'")
        sys.exit(1)
    
    update_website(
        version=sys.argv[1],
        quality_score=sys.argv[2],
        improvement_desc=sys.argv[3],
        pins=int(sys.argv[4]),
        lines=int(sys.argv[5]),
        gen_time=sys.argv[6],
        svg_filename=sys.argv[7],
        png_filename=sys.argv[8],
        params=sys.argv[9]
    )
