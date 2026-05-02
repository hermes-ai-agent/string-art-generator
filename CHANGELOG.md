# Changelog

All notable changes to String Art Generator will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-05-02

### Added
- **Edge detection preprocessing** using Sobel gradient for feature detection
- **Feature-aware line selection** - prioritizes edges and high-contrast areas
- **Edge weight parameter** - configurable multiplier for edge importance (default: 2.0)
- **Performance tracking** - generation time now logged in metadata
- **Improved metadata** in JSON output with version and improvements list

### Changed
- **Default max lines reduced** from 3000 to 2000 for better clarity
- **Line weight increased** from 20 to 30 for better visibility
- **Min distance increased** from 15 to 20 to reduce overlap
- **Algorithm improved** - now considers both darkness AND edge strength
- **Better quality** - less blur, clearer features, sharper edges

### Performance
- Generation time: ~150s for 2000 lines (was ~180s for 3000 lines)
- Better quality with fewer lines = win-win

### Fixed
- Blur issue from too many overlapping lines
- Poor feature recognition (eyes, nose, ears now clearer)
- Samar/unclear output - now much sharper

## [1.0.0] - 2026-05-02

### Added
- Initial release
- Circular pin arrangement
- Greedy line selection algorithm
- SVG output (600mm x 600mm, 0.18mm stroke)
- PNG preview generation
- JSON sequence data
- Text instructions for manual construction
- GitHub Pages showcase

### Known Issues
- Output too blurry with 3000 lines
- No edge detection - treats all areas equally
- Greedy algorithm finds local optimum, not global best
- Performance could be better
