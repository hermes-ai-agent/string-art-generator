# String Art Quality Issues - 2026-05-02

## Current Problems

### 1. Image Quality
- **Issue:** Hasil string art samar, kucing kurang jelas
- **Root Cause:** 
  - 3000 lines terlalu padat → overlap → blur effect
  - Greedy algorithm suboptimal → tidak prioritize important features
  - Line weight 0.18mm mungkin terlalu tipis untuk detail

### 2. Algorithm Limitations
- **Current:** Greedy line selection (pilih line terbaik per step)
- **Problem:** Local optimum, tidak consider global picture
- **Better approaches:**
  - TSP-based (Traveling Salesman Problem)
  - Genetic algorithms
  - Simulated annealing
  - Feature detection (prioritize edges, high-contrast areas)

### 3. Parameter Tuning Needed
- **Lines:** 3000 might be too many → try 1500-2000
- **Pins:** 200 might need adjustment → try 150 or 250
- **Line weight:** 0.18mm → try 0.25mm or 0.3mm for visibility
- **Min distance:** 15 → try different values

### 4. Image Preprocessing
- **Missing:** Edge detection, contrast enhancement
- **Should add:**
  - Canny edge detection
  - Adaptive thresholding
  - Feature prioritization (eyes, nose, ears for cat)

## Improvement Plan

### Phase 1: Algorithm Improvements (Self-Learning Focus)
1. Research TSP-based string art algorithms
2. Implement genetic algorithm variant
3. Add edge detection preprocessing
4. Feature-aware line selection

### Phase 2: Parameter Optimization
1. Grid search for optimal parameters
2. A/B testing different line counts
3. Adaptive line weight based on image complexity

### Phase 3: Performance (Go/Rust Rewrite)
1. Rewrite core algorithm in Go or Rust
2. Parallel processing for line selection
3. GPU acceleration for image processing
4. Target: <5 seconds for 2000 lines

## Expected Improvements
- **Clarity:** 3x better feature recognition
- **Speed:** 10-50x faster (Go/Rust)
- **Quality:** Sharper edges, better contrast
- **Flexibility:** Auto-tune parameters per image

## Self-Learning Priority
**HIGH** - This is the core quality issue that needs immediate attention.

Next learning sessions should focus on:
1. String art algorithm research (TSP, genetic)
2. Edge detection techniques
3. Feature-aware optimization
4. Performance optimization patterns
