# String Art Generator

Generate string art sequences from images using greedy line selection algorithm.

## Features

- **Circular pin arrangement** - Pins distributed evenly around a circle
- **Greedy algorithm** - Selects best line at each step to match target image
- **Multiple outputs**:
  - JSON sequence (for programmatic use)
  - Visual render (preview)
  - Comparison image (original vs result)
  - Text instructions (for manual construction)

## Installation

```bash
pip install pillow numpy --user
```

## Usage

Basic usage:
```bash
python3 string_art_generator.py input_image.jpg
```

Advanced options:
```bash
python3 string_art_generator.py input_image.jpg \
  --pins 300 \
  --lines 4000 \
  --min-distance 20 \
  --line-weight 25 \
  --output ./my_output/
```

### Parameters

- `--pins`: Number of pins around the circle (default: 200)
- `--lines`: Maximum number of strings (default: 3000)
- `--min-distance`: Minimum distance between consecutive pins (default: 15)
- `--line-weight`: Darkness contribution per string (default: 20)
- `--output`: Output directory (default: `string_art_output/` in image directory)

## Algorithm

1. **Setup**: Place N pins evenly around a circle
2. **Greedy Selection**: 
   - Start at pin 0
   - For each step, try all valid next pins
   - Select the pin that creates the line with maximum darkness match
   - Draw that line (lighten pixels along the path)
   - Move to the selected pin
3. **Termination**: Stop when max lines reached or no improvement possible

## Output Files

- `*_stringart_*.svg` - **MAIN OUTPUT** - SVG file (600mm x 600mm, 0.18mm stroke, opaque)
- `*_sequence_*.json` - Complete sequence data (pins + connections)
- `*_render_*.png` - PNG preview for visualization only
- `*_comparison_*.png` - Side-by-side original vs result
- `*_instructions_*.txt` - Human-readable pin-to-pin sequence

**IMPORTANT:** SVG is the primary output for physical construction. PNG files are previews only.

## Version History

### v1.0.0 (2026-05-02)
- Initial implementation
- Circular pin arrangement
- Greedy line selection algorithm
- Multiple output formats

## Future Improvements

This generator will be continuously improved through autonomous learning sessions:
- Advanced algorithms (TSP-based, genetic algorithms)
- Different pin arrangements (square, custom shapes)
- Multi-color string art
- Optimization techniques
- Better line rendering
- Physical construction considerations

## Learning Log

Improvements discovered through self-learning will be documented here:

---
*Last updated: 2026-05-02*
