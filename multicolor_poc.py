#!/usr/bin/env python3
"""
Multi-Color String Art Proof of Concept (v4.0.0-alpha)
Demonstrates CMY color model for string art

This is a simplified proof-of-concept to validate the multi-color approach
before integrating into the main generator.
"""

import numpy as np
from PIL import Image
import argparse


def rgb_to_cmy(rgb_array):
    """
    Convert RGB image to CMY color space
    CMY is subtractive color model, better for string art
    
    Args:
        rgb_array: RGB image array (H, W, 3) with values 0-255
    
    Returns:
        cmy_array: CMY image array (H, W, 3) with values 0-255
    """
    # Normalize to 0-1
    rgb_norm = rgb_array / 255.0
    
    # Convert to CMY: CMY = 1 - RGB
    cmy_norm = 1.0 - rgb_norm
    
    # Scale back to 0-255
    cmy_array = (cmy_norm * 255).astype(np.uint8)
    
    return cmy_array


def cmy_to_rgb(cmy_array):
    """
    Convert CMY image back to RGB
    
    Args:
        cmy_array: CMY image array (H, W, 3) with values 0-255
    
    Returns:
        rgb_array: RGB image array (H, W, 3) with values 0-255
    """
    # Normalize to 0-1
    cmy_norm = cmy_array / 255.0
    
    # Convert to RGB: RGB = 1 - CMY
    rgb_norm = 1.0 - cmy_norm
    
    # Scale back to 0-255
    rgb_array = (rgb_norm * 255).astype(np.uint8)
    
    return rgb_array


def visualize_color_channels(image_path, output_prefix="color_channels"):
    """
    Visualize RGB and CMY color channels separately
    This helps understand how multi-color string art would work
    """
    print(f"Loading image: {image_path}")
    img = Image.open(image_path).convert('RGB')
    
    # Resize for faster processing
    img = img.resize((400, 400), Image.Resampling.LANCZOS)
    
    # Convert to numpy
    rgb_array = np.array(img)
    
    print("Converting to CMY color space...")
    cmy_array = rgb_to_cmy(rgb_array)
    
    # Extract and save individual channels
    print("Extracting color channels...")
    
    # RGB channels
    r_channel = np.zeros_like(rgb_array)
    r_channel[:, :, 0] = rgb_array[:, :, 0]
    
    g_channel = np.zeros_like(rgb_array)
    g_channel[:, :, 1] = rgb_array[:, :, 1]
    
    b_channel = np.zeros_like(rgb_array)
    b_channel[:, :, 2] = rgb_array[:, :, 2]
    
    # CMY channels
    c_channel = np.zeros_like(cmy_array)
    c_channel[:, :, 0] = cmy_array[:, :, 0]  # Cyan
    
    m_channel = np.zeros_like(cmy_array)
    m_channel[:, :, 1] = cmy_array[:, :, 1]  # Magenta
    
    y_channel = np.zeros_like(cmy_array)
    y_channel[:, :, 2] = cmy_array[:, :, 2]  # Yellow
    
    # Save RGB channels
    Image.fromarray(r_channel).save(f"{output_prefix}_R.png")
    Image.fromarray(g_channel).save(f"{output_prefix}_G.png")
    Image.fromarray(b_channel).save(f"{output_prefix}_B.png")
    
    print(f"✓ Saved RGB channels: {output_prefix}_R.png, _G.png, _B.png")
    
    # Save CMY channels (convert back to RGB for display)
    Image.fromarray(cmy_to_rgb(c_channel)).save(f"{output_prefix}_C.png")
    Image.fromarray(cmy_to_rgb(m_channel)).save(f"{output_prefix}_M.png")
    Image.fromarray(cmy_to_rgb(y_channel)).save(f"{output_prefix}_Y.png")
    
    print(f"✓ Saved CMY channels: {output_prefix}_C.png, _M.png, _Y.png")
    
    # Save grayscale versions (for string art generation)
    c_gray = cmy_array[:, :, 0]
    m_gray = cmy_array[:, :, 1]
    y_gray = cmy_array[:, :, 2]
    
    Image.fromarray(c_gray).save(f"{output_prefix}_C_gray.png")
    Image.fromarray(m_gray).save(f"{output_prefix}_M_gray.png")
    Image.fromarray(y_gray).save(f"{output_prefix}_Y_gray.png")
    
    print(f"✓ Saved grayscale channels for string art: {output_prefix}_C_gray.png, _M_gray.png, _Y_gray.png")
    
    # Create comparison image
    comparison = Image.new('RGB', (400 * 4, 400 * 2))
    
    # Top row: Original, R, G, B
    comparison.paste(img, (0, 0))
    comparison.paste(Image.fromarray(r_channel), (400, 0))
    comparison.paste(Image.fromarray(g_channel), (800, 0))
    comparison.paste(Image.fromarray(b_channel), (1200, 0))
    
    # Bottom row: CMY conversion, C, M, Y
    comparison.paste(Image.fromarray(cmy_to_rgb(cmy_array)), (0, 400))
    comparison.paste(Image.fromarray(cmy_to_rgb(c_channel)), (400, 400))
    comparison.paste(Image.fromarray(cmy_to_rgb(m_channel)), (800, 400))
    comparison.paste(Image.fromarray(cmy_to_rgb(y_channel)), (1200, 400))
    
    comparison.save(f"{output_prefix}_comparison.png")
    print(f"✓ Saved comparison: {output_prefix}_comparison.png")
    
    print("\n" + "="*60)
    print("Multi-Color String Art Concept:")
    print("="*60)
    print("1. Convert RGB image to CMY color space")
    print("2. Extract C, M, Y channels as grayscale images")
    print("3. Generate string art for each channel separately:")
    print("   - Cyan strings for C channel")
    print("   - Magenta strings for M channel")
    print("   - Yellow strings for Y channel")
    print("4. Overlay all three colored string arts")
    print("5. Result: Full-color string art through subtractive color mixing")
    print("="*60)
    
    return {
        'rgb_channels': [r_channel, g_channel, b_channel],
        'cmy_channels': [c_channel, m_channel, y_channel],
        'cmy_gray': [c_gray, m_gray, y_gray]
    }


def main():
    parser = argparse.ArgumentParser(description='Multi-Color String Art Proof of Concept')
    parser.add_argument('image', help='Input RGB image path')
    parser.add_argument('--output-prefix', default='color_channels', help='Output file prefix')
    
    args = parser.parse_args()
    
    visualize_color_channels(args.image, args.output_prefix)
    
    print("\nNext steps:")
    print("1. Use the grayscale channel images (_C_gray.png, _M_gray.png, _Y_gray.png)")
    print("2. Generate string art for each using the main generator:")
    print("   python3 string_art_generator.py color_channels_C_gray.png --pins 200 --lines 1000")
    print("   python3 string_art_generator.py color_channels_M_gray.png --pins 200 --lines 1000")
    print("   python3 string_art_generator.py color_channels_Y_gray.png --pins 200 --lines 1000")
    print("3. Combine the three SVG outputs with cyan, magenta, yellow colors")
    print("4. Result: Multi-color string art!")


if __name__ == '__main__':
    main()
