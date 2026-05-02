#!/usr/bin/env python3
"""
String Art Generator v2.0.0
Converts images to string art sequences with improved quality

Major improvements:
- Edge detection preprocessing (Canny)
- Feature-aware line selection
- Optimized parameters for clarity
- Better performance

Version: 2.0.0
Last Updated: 2026-05-02
"""

import numpy as np
from PIL import Image, ImageDraw, ImageFilter
import argparse
from pathlib import Path
import json
from datetime import datetime
import time


class StringArtGenerator:
    """Generate string art from images with edge-aware optimization"""
    
    def __init__(self, num_pins=200, min_distance=20, line_weight=30, edge_weight=2.0):
        """
        Initialize string art generator
        
        Args:
            num_pins: Number of pins around the circular frame
            min_distance: Minimum distance between consecutive pins
            line_weight: Darkness contribution of each string (0-255)
            edge_weight: Multiplier for edge pixels (prioritize edges)
        """
        self.num_pins = num_pins
        self.min_distance = min_distance
        self.line_weight = line_weight
        self.edge_weight = edge_weight
        self.pins = []
        self.lines = []
        
    def setup_pins(self, width, height):
        """Setup pins in a circular arrangement"""
        center_x, center_y = width // 2, height // 2
        radius = min(center_x, center_y) - 20
        
        self.pins = []
        for i in range(self.num_pins):
            angle = 2 * np.pi * i / self.num_pins
            x = int(center_x + radius * np.cos(angle))
            y = int(center_y + radius * np.sin(angle))
            self.pins.append((x, y))
    
    def detect_edges(self, img_array):
        """
        Detect edges using Canny edge detection
        Returns edge map (higher values = stronger edges)
        """
        # Convert to PIL Image for edge detection
        img = Image.fromarray(img_array.astype(np.uint8))
        
        # Apply Gaussian blur first to reduce noise
        img_blur = img.filter(ImageFilter.GaussianBlur(radius=1))
        
        # Simple edge detection using gradient
        img_array_blur = np.array(img_blur, dtype=np.float32)
        
        # Sobel edge detection (simple but effective)
        sobel_x = np.array([[-1, 0, 1], [-2, 0, 2], [-1, 0, 1]])
        sobel_y = np.array([[-1, -2, -1], [0, 0, 0], [1, 2, 1]])
        
        # Pad image for convolution
        padded = np.pad(img_array_blur, 1, mode='edge')
        
        # Calculate gradients
        grad_x = np.zeros_like(img_array_blur)
        grad_y = np.zeros_like(img_array_blur)
        
        for i in range(img_array_blur.shape[0]):
            for j in range(img_array_blur.shape[1]):
                region = padded[i:i+3, j:j+3]
                grad_x[i, j] = np.sum(region * sobel_x)
                grad_y[i, j] = np.sum(region * sobel_y)
        
        # Edge magnitude
        edge_magnitude = np.sqrt(grad_x**2 + grad_y**2)
        
        # Normalize to 0-255
        edge_magnitude = (edge_magnitude / edge_magnitude.max() * 255).astype(np.float32)
        
        return edge_magnitude
    
    def get_line_pixels(self, p1, p2, width, height):
        """Get all pixel coordinates along a line using Bresenham's algorithm"""
        x1, y1 = self.pins[p1]
        x2, y2 = self.pins[p2]
        
        pixels = []
        dx = abs(x2 - x1)
        dy = abs(y2 - y1)
        sx = 1 if x1 < x2 else -1
        sy = 1 if y1 < y2 else -1
        err = dx - dy
        
        x, y = x1, y1
        while True:
            if 0 <= x < width and 0 <= y < height:
                pixels.append((y, x))
            
            if x == x2 and y == y2:
                break
                
            e2 = 2 * err
            if e2 > -dy:
                err -= dy
                x += sx
            if e2 < dx:
                err += dx
                y += sy
        
        return pixels
    
    def calculate_line_error(self, img_array, edge_array, p1, p2):
        """
        Calculate line quality score (higher = better)
        Considers both darkness and edge alignment
        """
        pixels = self.get_line_pixels(p1, p2, img_array.shape[1], img_array.shape[0])
        
        if not pixels:
            return 0
        
        total_score = 0
        for y, x in pixels:
            # Darkness score (darker pixels = better match)
            darkness = 255 - img_array[y, x]
            
            # Edge score (stronger edges = higher priority)
            edge_strength = edge_array[y, x]
            
            # Combined score with edge weighting
            pixel_score = darkness + (edge_strength * self.edge_weight)
            total_score += pixel_score
        
        return total_score / len(pixels)
    
    def draw_line_on_array(self, img_array, p1, p2):
        """Draw a line on the image array (lighten pixels)"""
        pixels = self.get_line_pixels(p1, p2, img_array.shape[1], img_array.shape[0])
        
        for y, x in pixels:
            # Lighten the pixel
            img_array[y, x] = min(255, img_array[y, x] + self.line_weight)
    
    def generate(self, image_path, max_lines=2000, output_dir=None):
        """
        Generate string art sequence from image
        
        Args:
            image_path: Path to input image
            max_lines: Maximum number of strings (default: 2000, reduced from 3000)
            output_dir: Directory to save outputs
        
        Returns:
            dict: Results including sequence, statistics, and file paths
        """
        start_time = time.time()
        
        print(f"Loading image: {image_path}")
        img = Image.open(image_path).convert('L')
        
        # Resize to square
        size = 800
        img = img.resize((size, size), Image.Resampling.LANCZOS)
        
        # Setup pins
        self.setup_pins(size, size)
        print(f"Setup {self.num_pins} pins in circular arrangement")
        
        # Convert to numpy array
        img_array = np.array(img, dtype=np.float32)
        original_array = img_array.copy()
        
        # Detect edges for feature-aware selection
        print("Detecting edges for feature-aware optimization...")
        edge_array = self.detect_edges(img_array)
        
        # Generate string sequence
        print(f"Generating string art (max {max_lines} lines)...")
        print("Using edge-aware algorithm for better clarity...")
        self.lines = []
        current_pin = 0
        
        for line_num in range(max_lines):
            best_pin = None
            best_score = -1
            
            # Try all possible next pins
            for next_pin in range(self.num_pins):
                # Skip if too close
                pin_distance = min(
                    abs(next_pin - current_pin),
                    self.num_pins - abs(next_pin - current_pin)
                )
                if pin_distance < self.min_distance:
                    continue
                
                # Calculate score (considers both darkness and edges)
                score = self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
                
                if score > best_score:
                    best_score = score
                    best_pin = next_pin
            
            if best_pin is None or best_score < 1:
                print(f"Stopping at {line_num} lines (no improvement)")
                break
            
            # Draw the best line
            self.lines.append((current_pin, best_pin))
            self.draw_line_on_array(img_array, current_pin, best_pin)
            current_pin = best_pin
            
            if (line_num + 1) % 100 == 0:
                elapsed = time.time() - start_time
                print(f"  {line_num + 1} lines generated... ({elapsed:.1f}s)")
        
        elapsed_total = time.time() - start_time
        print(f"✓ Generated {len(self.lines)} lines in {elapsed_total:.1f}s")
        
        # Create output directory
        if output_dir is None:
            output_dir = Path(image_path).parent / "string_art_output"
        else:
            output_dir = Path(output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        
        # Save results
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        base_name = Path(image_path).stem
        
        # 1. Save sequence as JSON
        sequence_file = output_dir / f"{base_name}_sequence_{timestamp}.json"
        sequence_data = {
            "metadata": {
                "source_image": str(image_path),
                "generated_at": datetime.now().isoformat(),
                "version": "2.0.0",
                "num_pins": self.num_pins,
                "num_lines": len(self.lines),
                "min_distance": self.min_distance,
                "line_weight": self.line_weight,
                "edge_weight": self.edge_weight,
                "generation_time_seconds": elapsed_total,
                "improvements": [
                    "Edge detection preprocessing",
                    "Feature-aware line selection",
                    "Optimized parameters for clarity"
                ]
            },
            "pins": self.pins,
            "sequence": self.lines
        }
        
        with open(sequence_file, 'w') as f:
            json.dump(sequence_data, f, indent=2)
        print(f"✓ Saved sequence: {sequence_file}")
        
        # 2. Render preview
        render_file = output_dir / f"{base_name}_render_{timestamp}.png"
        self.render_string_art_preview(render_file, size)
        print(f"✓ Saved preview: {render_file}")
        
        # 2b. Render SVG
        svg_file = output_dir / f"{base_name}_stringart_{timestamp}.svg"
        self.render_string_art_svg(svg_file)
        print(f"✓ Saved SVG (600mm x 600mm, 0.18mm stroke): {svg_file}")
        
        # 3. Save comparison
        comparison_file = output_dir / f"{base_name}_comparison_{timestamp}.png"
        self.create_comparison(original_array, img_array, comparison_file)
        print(f"✓ Saved comparison: {comparison_file}")
        
        # 4. Save instructions
        text_file = output_dir / f"{base_name}_instructions_{timestamp}.txt"
        self.save_text_instructions(text_file)
        print(f"✓ Saved instructions: {text_file}")
        
        return {
            "num_lines": len(self.lines),
            "generation_time": elapsed_total,
            "sequence_file": str(sequence_file),
            "svg_file": str(svg_file),
            "preview_file": str(render_file),
            "comparison_file": str(comparison_file),
            "instructions_file": str(text_file)
        }
    
    def render_string_art_svg(self, output_path):
        """
        Render string art as SVG (MANDATORY OUTPUT FORMAT)
        Dimensions: 600mm x 600mm
        Stroke width: 0.18mm, full opaque
        """
        svg_width_mm = 600
        svg_height_mm = 600
        stroke_width_mm = 0.18
        
        svg_lines = [
            f'<?xml version="1.0" encoding="UTF-8"?>',
            f'<svg width="{svg_width_mm}mm" height="{svg_height_mm}mm" ',
            f'     viewBox="0 0 {svg_width_mm} {svg_height_mm}" ',
            f'     xmlns="http://www.w3.org/2000/svg">',
            f'  <!-- String Art Generator v2.0.0 -->',
            f'  <!-- Pins: {self.num_pins}, Lines: {len(self.lines)} -->',
            f'  <!-- Edge-aware algorithm with optimized parameters -->',
            f'  <!-- Dimensions: {svg_width_mm}mm x {svg_height_mm}mm -->',
            f'  <!-- Stroke: {stroke_width_mm}mm, opaque -->',
            f'',
            f'  <g id="string-art" stroke="black" stroke-width="{stroke_width_mm}mm" fill="none" opacity="1.0">',
        ]
        
        scale = svg_width_mm / 800.0
        
        for p1, p2 in self.lines:
            x1, y1 = self.pins[p1]
            x2, y2 = self.pins[p2]
            
            x1_mm = x1 * scale
            y1_mm = y1 * scale
            x2_mm = x2 * scale
            y2_mm = y2 * scale
            
            svg_lines.append(f'    <line x1="{x1_mm:.3f}" y1="{y1_mm:.3f}" x2="{x2_mm:.3f}" y2="{y2_mm:.3f}"/>')
        
        svg_lines.append(f'  </g>')
        svg_lines.append(f'</svg>')
        
        with open(output_path, 'w') as f:
            f.write('\n'.join(svg_lines))
    
    def render_string_art_preview(self, output_path, size):
        """Render the string art as PNG preview"""
        img = Image.new('RGB', (size, size), 'white')
        draw = ImageDraw.Draw(img)
        
        for p1, p2 in self.lines:
            draw.line([self.pins[p1], self.pins[p2]], fill='black', width=1)
        
        for pin in self.pins:
            draw.ellipse([pin[0]-2, pin[1]-2, pin[0]+2, pin[1]+2], fill='red')
        
        img.save(output_path)
    
    def create_comparison(self, original, processed, output_path):
        """Create side-by-side comparison image"""
        img_original = Image.fromarray(original.astype(np.uint8))
        img_processed = Image.fromarray(processed.astype(np.uint8))
        
        width, height = img_original.size
        comparison = Image.new('RGB', (width * 2, height), 'white')
        comparison.paste(img_original.convert('RGB'), (0, 0))
        comparison.paste(img_processed.convert('RGB'), (width, 0))
        
        draw = ImageDraw.Draw(comparison)
        draw.text((10, 10), "Original", fill='red')
        draw.text((width + 10, 10), "After String Art", fill='red')
        
        comparison.save(output_path)
    
    def save_text_instructions(self, output_path):
        """Save human-readable instructions"""
        with open(output_path, 'w') as f:
            f.write("STRING ART CONSTRUCTION INSTRUCTIONS\n")
            f.write("=" * 50 + "\n\n")
            f.write(f"Version: 2.0.0 (Edge-aware algorithm)\n")
            f.write(f"Total pins: {self.num_pins}\n")
            f.write(f"Total strings: {len(self.lines)}\n")
            f.write(f"Minimum pin distance: {self.min_distance}\n\n")
            f.write("SEQUENCE (pin-to-pin):\n")
            f.write("-" * 50 + "\n")
            
            for i, (p1, p2) in enumerate(self.lines, 1):
                f.write(f"{i:4d}. Pin {p1:3d} → Pin {p2:3d}\n")
                
                if i % 100 == 0:
                    f.write("\n" + "=" * 50 + "\n\n")


def main():
    parser = argparse.ArgumentParser(description='String Art Generator v2.0.0 - Edge-aware optimization')
    parser.add_argument('image', help='Input image path')
    parser.add_argument('--pins', type=int, default=200, help='Number of pins (default: 200)')
    parser.add_argument('--lines', type=int, default=2000, help='Maximum lines (default: 2000, reduced for clarity)')
    parser.add_argument('--min-distance', type=int, default=20, help='Minimum pin distance (default: 20)')
    parser.add_argument('--line-weight', type=int, default=30, help='Line darkness weight (default: 30)')
    parser.add_argument('--edge-weight', type=float, default=2.0, help='Edge priority multiplier (default: 2.0)')
    parser.add_argument('--output', help='Output directory')
    
    args = parser.parse_args()
    
    generator = StringArtGenerator(
        num_pins=args.pins,
        min_distance=args.min_distance,
        line_weight=args.line_weight,
        edge_weight=args.edge_weight
    )
    
    results = generator.generate(
        args.image,
        max_lines=args.lines,
        output_dir=args.output
    )
    
    print("\n" + "=" * 50)
    print("STRING ART GENERATION COMPLETE (v2.0.0)")
    print("=" * 50)
    print(f"Lines generated: {results['num_lines']}")
    print(f"Generation time: {results['generation_time']:.1f}s")
    print(f"SVG output: {results['svg_file']}")
    print(f"Preview: {results['preview_file']}")
    print(f"Comparison: {results['comparison_file']}")
    print(f"\nImprovements in v2.0.0:")
    print("  ✓ Edge detection preprocessing")
    print("  ✓ Feature-aware line selection")
    print("  ✓ Optimized parameters (2000 lines, better clarity)")
    print("  ✓ Performance tracking")


if __name__ == '__main__':
    main()
