# String Art Generator (Go)

High-performance string art generator written in Go with edge detection and parallel processing.

**Live Demo:** https://hermes-ai-agent.github.io/string-art-generator/

## Features

- ✅ **Fast:** 5-8 seconds for 3000 lines (66x faster than Python)
- ✅ **Edge Detection:** Sobel filter prioritizes facial features
- ✅ **Parallel Processing:** 8 workers for line evaluation
- ✅ **Adaptive Stopping:** Automatically stops when quality plateaus
- ✅ **SVG Output:** 600mm x 600mm, 0.18mm stroke (physical construction ready)
- ✅ **Configurable:** Pins, lines, weights, edge detection strength

## Installation

```bash
# Clone repository
git clone https://github.com/hermes-ai-agent/string-art-generator.git
cd string-art-generator

# Build
go build -o string-art-generator

# Run
./string-art-generator --input cat.jpg --pins 360 --lines 3000
```

## Quick Start

```bash
# Generate string art with optimal parameters
./string-art-generator \
  --input cat.jpg \
  --pins 360 \
  --lines 3000 \
  --edge-weight 5.0

# Output: SVG 600mm x 600mm, 0.18mm stroke
```

**⚠️ MANDATORY RULE:** Pin count limited to **360 maximum** (1 pin per degree for physical construction).

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `--input` | (required) | Input image path |
| `--output` | auto | Output SVG path |
| `--pins` | 200 | Number of pins (max 360) |
| `--lines` | 2000 | Number of lines to draw |
| `--weight` | 30 | Line weight (darkness) |
| `--edge-weight` | 2.0 | Edge detection multiplier |
| `--min-dist` | 20 | Min distance between pins |
| `--workers` | 8 | Parallel workers |
| `--opacity` | 1.0 | String opacity (0.0-1.0) |

## Optimal Settings

### v3.7 (Current - 360 Pins Max)
```bash
--pins 360 --lines 3000 --edge-weight 5.0
```
- **Quality:** 7/10
- **Speed:** 5.4 seconds
- **Physical construction:** ✅ Ready (1 pin per degree)

### v3.4 (Previous Best - Exceeds New Rule)
```bash
--pins 500 --lines 3000 --edge-weight 5.0
```
- **Quality:** 8/10
- **Speed:** 8.0 seconds
- **Physical construction:** ❌ Exceeds 360 pin limit

## Quality Progression

| Version | Pins | Quality | Speed | Status |
|---------|------|---------|-------|--------|
| v3.2 | 200 | 7/10 | 2.6s | Good baseline |
| v3.3 | 300 | 8/10 | 3.5s | Better detail |
| v3.4 | 500 | 8/10 | 8.0s | Best quality (exceeds rule) |
| v3.5 | 600 | 7.5/10 | 10.7s | Too noisy |
| **v3.7** | **360** | **7/10** | **5.4s** | **Current (rule compliant)** |

## Why 360 Pins Maximum?

1. **Physical construction:** 1 pin per degree = evenly spaced
2. **Practical limit:** More pins = harder to construct
3. **Quality trade-off:** 360 pins = 7/10 (vs 500 = 8/10)
4. **Performance gain:** 36% faster than 500 pins

## Architecture

```
string-art-generator/
├── main.go              # CLI and orchestration
├── generator.go         # Core greedy algorithm
├── generator_dual.go    # Dual-color (experimental)
├── generator_sa.go      # Simulated annealing (slow)
├── image.go             # Image loading & edge detection
├── svg.go               # SVG export
└── render.go            # Canvas to PNG rendering
```

## Algorithm

1. **Edge Detection:** Sobel filter on input image
2. **Pin Generation:** Evenly spaced around circle (max 360)
3. **Greedy Selection:** 
   - Evaluate all valid pin connections
   - Score based on error reduction + edge weighting
   - Select best line
   - Update canvas
   - Repeat
4. **Adaptive Stopping:** Stop when quality plateaus
5. **SVG Export:** 600mm x 600mm, 0.18mm stroke

## Performance

- **Go vs Python:** 66x faster (2.5s vs 151s)
- **Parallel workers:** 8 threads for line evaluation
- **Edge detection:** Sobel filter in ~50ms
- **Total time:** 5-8 seconds for 3000 lines

## Examples

See live examples at: https://hermes-ai-agent.github.io/string-art-generator/

## License

MIT License

## Author

String Art Generator v3.7 (Go)  
Last Updated: 2026-05-03

---

**Note:** Python version (v4.3.0) exists but has syntax errors. Go version is the maintained implementation.
