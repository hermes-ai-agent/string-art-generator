#!/usr/bin/env python3
"""
Render SVG to PNG using cairosvg
"""
import sys
import subprocess

def render_svg_to_png(svg_path, png_path, width=800, height=800):
    """Render SVG to PNG using cairosvg"""
    try:
        import cairosvg
        cairosvg.svg2png(
            url=svg_path,
            write_to=png_path,
            output_width=width,
            output_height=height
        )
        print(f"✓ Rendered {svg_path} → {png_path}")
        return True
    except ImportError:
        print("Installing cairosvg...")
        subprocess.run([sys.executable, "-m", "pip", "install", "cairosvg"], check=True)
        import cairosvg
        cairosvg.svg2png(
            url=svg_path,
            write_to=png_path,
            output_width=width,
            output_height=height
        )
        print(f"✓ Rendered {svg_path} → {png_path}")
        return True

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: python3 render_svg.py <input.svg> <output.png> [width] [height]")
        sys.exit(1)
    
    svg_path = sys.argv[1]
    png_path = sys.argv[2]
    width = int(sys.argv[3]) if len(sys.argv) > 3 else 800
    height = int(sys.argv[4]) if len(sys.argv) > 4 else 800
    
    render_svg_to_png(svg_path, png_path, width, height)
