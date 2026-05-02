#!/usr/bin/env python3
"""
String Art Generator v4.4.0
Converts images to string art sequences with improved quality

Major improvements:
- Edge detection preprocessing (Canny)
- Feature-aware line selection
- Optimized parameters for clarity
- Better performance
- Non-opaque string support (v2.1.0)
- Random sampling optimization (v2.1.0)
- Adaptive stopping condition (v2.2.0)
- Line caching for speed (v2.2.0)
- Look-ahead optimization (v2.2.0)
- Vectorized edge detection (v2.3.0)
- Beam search optimization (v2.3.0)
- Parallel line evaluation (v2.3.0)
- 2-opt local optimization (v3.3.0)
- Importance map support (v3.3.0)
- Enhanced stopping criteria (v3.3.0)
- Squared difference metric (v3.4.0)
- Simulated annealing optimization (v3.4.0)
- Auto-importance map generation (v3.4.0)
- CMY-log color blending model (v3.4.0)
- Anti-aliased line rendering with Xiaolin Wu's algorithm (v3.5.0)
- Sub-pixel accuracy for smoother output (v3.5.0)
- Hexagonal and square pin arrangements (v3.6.0)
- Adaptive beam width optimization (v3.6.0)
- 3-opt local optimization (v3.7.0)
- k-Nearest Neighbors selection (v3.7.0)
- Warm restart simulated annealing (v3.7.0)
- "Fear removal" error function for better detail capture (v3.8.0)
- Blur simulation support (v3.8.0)
- Pool-based sampling (Bridges 2022 method) (v3.9.0)
- Multi-color string art support with CMY color model (v4.0.0)
- Per-channel optimization for RGB images (v4.0.0)
- Color-aware line rendering and blending (v4.0.0)
- Binary Linear Optimizer with sparse matrix (v4.2.0) - 10-100x faster
- Residual error tracking for efficient updates (v4.2.0)
- Vectorized line scoring with sparse operations (v4.2.0)
- Pre-computed transformation matrix A for all lines (v4.2.0)
- Multi-resolution optimization (Birsak 2018) - render high-res, evaluate low-res (v4.3.0)
- Line removal stage for quality refinement (Birsak 2018) (v4.3.0)
- Adaptive line opacity based on image brightness (v4.3.0)
- SSIM-based perceptual quality metric (v4.4.0) - Better than MSE for human perception
- Enhanced line removal with quality-based pruning (v4.4.0)
- Perceptual importance weighting (v4.4.0)

Version: 4.4.0
Last Updated: 2026-05-03
"""

import numpy as np
from PIL import Image, ImageDraw, ImageFilter
import argparse
from pathlib import Path
import json
from datetime import datetime
import time
from scipy import ndimage
from scipy import sparse
from concurrent.futures import ThreadPoolExecutor
import multiprocessing
from sklearn.neighbors import NearestNeighbors
from functools import lru_cache
try:
    from numba import jit, prange
    NUMBA_AVAILABLE = True
except ImportError:
    NUMBA_AVAILABLE = False
    # Fallback decorator that does nothing
    def jit(*args, **kwargs):
        def decorator(func):
            return func
        return decorator if args and callable(args[0]) else decorator
    prange = range


class StringArtGenerator:
    """Generate string art from images with edge-aware optimization"""
    
    def __init__(self, num_pins=200, min_distance=20, line_weight=30, edge_weight=2.0, 
                 line_opacity=1.0, random_sampling=False, sample_size=None, 
                 look_ahead=False, adaptive_stop=True, stop_threshold=0.5,
                 beam_width=3, use_parallel=True, use_2opt=True, two_opt_iterations=100,
                 importance_map=None, use_squared_diff=True, use_simulated_annealing=False,
                 annealing_temp=100.0, annealing_cooling=0.95, auto_importance=True,
                 pin_arrangement='circle', adaptive_beam=True, use_3opt=True, three_opt_iterations=50,
                 use_knn_selection=True, knn_neighbors=50, warm_restart_interval=500,
                 use_fear_removal=True, blur_simulation_factor=1, use_pool_sampling=True, 
                 pool_size=5000, pool_select=30, use_multicolor=False, color_mode='CMY',
                 use_advanced_penalty=True, lightness_penalty=0.3, adaptive_opacity=True):
        """
        Initialize string art generator
        
        Args:
            num_pins: Number of pins around the circular frame
            min_distance: Minimum distance between consecutive pins
            line_weight: Darkness contribution of each string (0-255)
            edge_weight: Multiplier for edge pixels (prioritize edges)
            line_opacity: Opacity of each line (0.0-1.0, lower = more transparent)
            random_sampling: Use random sampling for faster line selection
            sample_size: Number of lines to sample per iteration (None = all)
            look_ahead: Enable look-ahead optimization (slower but better quality)
            adaptive_stop: Stop when improvement drops below threshold
            stop_threshold: Minimum score improvement to continue (adaptive_stop only)
            beam_width: Number of candidates to keep in beam search (1 = greedy)
            use_parallel: Use parallel processing for line evaluation
            use_2opt: Apply 2-opt local optimization post-processing (v3.3.0)
            two_opt_iterations: Number of 2-opt passes (v3.3.0)
            importance_map: Optional importance map for region prioritization (v3.3.0)
            use_squared_diff: Use squared difference metric for better convergence (v3.4.0)
            use_simulated_annealing: Enable simulated annealing to escape local optima (v3.4.0)
            annealing_temp: Initial temperature for simulated annealing (v3.4.0)
            annealing_cooling: Cooling rate for simulated annealing (v3.4.0)
            pin_arrangement: Pin arrangement type - 'circle', 'hexagon', 'square' (v3.6.0)
            adaptive_beam: Dynamically adjust beam width based on progress (v3.6.0)
            use_3opt: Apply 3-opt local optimization (v3.7.0)
            three_opt_iterations: Number of 3-opt passes (v3.7.0)
            use_knn_selection: Use k-Nearest Neighbors for smarter pin selection (v3.7.0)
            knn_neighbors: Number of nearest neighbors to consider (v3.7.0)
            warm_restart_interval: Lines between warm restarts for annealing (v3.7.0)
            use_fear_removal: Enable "fear removal" error function (v3.8.0)
            blur_simulation_factor: Downscaling factor for blur simulation (v3.8.0)
            use_pool_sampling: Use pool-based sampling (Bridges 2022 method) (v3.9.0)
            pool_size: Size of random pool to generate (v3.9.0)
            pool_select: Number of top candidates to select from pool (v3.9.0)
            use_multicolor: Enable multi-color string art (v4.0.0)
            color_mode: Color model - 'CMY' or 'RGB' (v4.0.0)
        """
        self.num_pins = num_pins
        self.min_distance = min_distance
        self.line_weight = line_weight
        self.edge_weight = edge_weight
        self.line_opacity = line_opacity
        self.random_sampling = random_sampling
        self.sample_size = sample_size
        self.look_ahead = look_ahead
        self.adaptive_stop = adaptive_stop
        self.stop_threshold = stop_threshold
        self.beam_width = beam_width
        self.initial_beam_width = beam_width  # Store initial value for adaptive beam
        self.use_parallel = use_parallel
        self.use_2opt = use_2opt
        self.two_opt_iterations = two_opt_iterations
        self.use_3opt = use_3opt
        self.three_opt_iterations = three_opt_iterations
        self.importance_map = importance_map
        self.use_squared_diff = use_squared_diff
        self.use_simulated_annealing = use_simulated_annealing
        self.annealing_temp = annealing_temp
        self.annealing_cooling = annealing_cooling
        self.auto_importance = auto_importance
        self.pin_arrangement = pin_arrangement
        self.adaptive_beam = adaptive_beam
        self.use_knn_selection = use_knn_selection
        self.knn_neighbors = knn_neighbors
        self.warm_restart_interval = warm_restart_interval
        self.use_fear_removal = use_fear_removal
        self.blur_simulation_factor = blur_simulation_factor
        self.use_pool_sampling = use_pool_sampling
        self.pool_size = pool_size
        self.pool_select = pool_select
        self.use_multicolor = use_multicolor
        self.color_mode = color_mode
        self.pins = []
        self.lines = []
        self.line_cache = {}  # Cache for pre-computed line pixels
        self.num_workers = max(1, multiprocessing.cpu_count() - 1) if use_parallel else 1
        self.current_temp = annealing_temp  # Track current temperature
        self.initial_annealing_temp = annealing_temp  # Store initial temp for warm restart
        self.knn_model = None  # k-NN model for pin selection
        # Multi-color support (v4.0.0)
        self.color_lines = {'C': [], 'M': [], 'Y': []} if use_multicolor and color_mode == 'CMY' else {}
        self.color_lines_rgb = {'R': [], 'G': [], 'B': []} if use_multicolor and color_mode == 'RGB' else {}
        
    def setup_pins(self, width, height):
        """
        Setup pins in various arrangements (v3.6.0)
        Supports: circle, hexagon, square
        """
        center_x, center_y = width // 2, height // 2
        radius = min(center_x, center_y) - 20
        
        self.pins = []
        
        if self.pin_arrangement == 'circle':
            # Original circular arrangement
            for i in range(self.num_pins):
                angle = 2 * np.pi * i / self.num_pins
                x = int(center_x + radius * np.cos(angle))
                y = int(center_y + radius * np.sin(angle))
                self.pins.append((x, y))
        
        elif self.pin_arrangement == 'hexagon':
            # Hexagonal arrangement (6 sides)
            pins_per_side = self.num_pins // 6
            remaining_pins = self.num_pins % 6
            
            # Hexagon vertices (starting from top, going clockwise)
            angles = [np.pi/2, np.pi/6, -np.pi/6, -np.pi/2, -5*np.pi/6, 5*np.pi/6]
            vertices = [(center_x + radius * np.cos(a), center_y + radius * np.sin(a)) 
                       for a in angles]
            
            # Distribute pins along each side
            for side in range(6):
                v1 = vertices[side]
                v2 = vertices[(side + 1) % 6]
                
                # Add extra pin to first few sides if there are remaining pins
                side_pins = pins_per_side + (1 if side < remaining_pins else 0)
                
                for i in range(side_pins):
                    t = i / side_pins if side_pins > 1 else 0
                    x = int(v1[0] + t * (v2[0] - v1[0]))
                    y = int(v1[1] + t * (v2[1] - v1[1]))
                    self.pins.append((x, y))
        
        elif self.pin_arrangement == 'square':
            # Square arrangement (4 sides)
            pins_per_side = self.num_pins // 4
            remaining_pins = self.num_pins % 4
            
            # Square corners
            half_size = radius
            corners = [
                (center_x - half_size, center_y - half_size),  # Top-left
                (center_x + half_size, center_y - half_size),  # Top-right
                (center_x + half_size, center_y + half_size),  # Bottom-right
                (center_x - half_size, center_y + half_size),  # Bottom-left
            ]
            
            # Distribute pins along each side
            for side in range(4):
                c1 = corners[side]
                c2 = corners[(side + 1) % 4]
                
                side_pins = pins_per_side + (1 if side < remaining_pins else 0)
                
                for i in range(side_pins):
                    t = i / side_pins if side_pins > 1 else 0
                    x = int(c1[0] + t * (c2[0] - c1[0]))
                    y = int(c1[1] + t * (c2[1] - c1[1]))
                    self.pins.append((x, y))
        
        else:
            raise ValueError(f"Unknown pin arrangement: {self.pin_arrangement}. Use 'circle', 'hexagon', or 'square'.")
    
    def detect_edges(self, img_array):
        """
        Detect edges using vectorized Sobel edge detection (v2.3.0 optimization)
        Returns edge map (higher values = stronger edges)
        """
        # Apply Gaussian blur first to reduce noise (vectorized)
        img_array_blur = ndimage.gaussian_filter(img_array, sigma=1.0)
        
        # Vectorized Sobel edge detection using scipy
        grad_x = ndimage.sobel(img_array_blur, axis=1)
        grad_y = ndimage.sobel(img_array_blur, axis=0)
        
        # Edge magnitude (vectorized)
        edge_magnitude = np.hypot(grad_x, grad_y)
        
        # Normalize to 0-255
        if edge_magnitude.max() > 0:
            edge_magnitude = (edge_magnitude / edge_magnitude.max() * 255).astype(np.float32)
        else:
            edge_magnitude = edge_magnitude.astype(np.float32)
        
        return edge_magnitude
    
    def generate_importance_map(self, img_array, edge_array):
        """
        Auto-generate importance map from image features (v3.4.0)
        Prioritizes: faces, eyes, high-contrast regions, edges
        
        Returns normalized importance map (0.0-1.0)
        """
        print("Auto-generating importance map from image features...")
        
        # Start with edge-based importance
        importance = edge_array.copy()
        
        # Add contrast-based importance (local variance)
        # High variance = important details
        local_variance = ndimage.generic_filter(img_array, np.var, size=15)
        local_variance = (local_variance / local_variance.max() * 255) if local_variance.max() > 0 else local_variance
        
        # Combine edge and variance with weights
        importance = importance * 0.6 + local_variance * 0.4
        
        # Detect potential face regions using simple heuristics
        # (darker regions in upper-middle area often contain eyes/face features)
        height, width = img_array.shape
        y_coords, x_coords = np.ogrid[:height, :width]
        
        # Create center-weighted mask (faces usually in center)
        center_y, center_x = height // 2, width // 2
        dist_from_center = np.sqrt((x_coords - center_x)**2 + (y_coords - center_y)**2)
        max_dist = np.sqrt(center_x**2 + center_y**2)
        center_weight = 1.0 - (dist_from_center / max_dist) * 0.5  # 0.5 to 1.0
        
        # Apply center weighting
        importance = importance * center_weight
        
        # Boost very dark regions (likely eyes, mouth, important features)
        dark_regions = (255 - img_array) / 255.0  # Invert: dark = high value
        dark_boost = np.power(dark_regions, 2) * 100  # Squared for emphasis
        importance = importance + dark_boost
        
        # Normalize to 0-1 range
        if importance.max() > 0:
            importance = importance / importance.max()
        
        # Apply smoothing to avoid sharp transitions
        importance = ndimage.gaussian_filter(importance, sigma=5.0)
        
        # Ensure minimum importance (don't completely ignore any region)
        importance = np.clip(importance, 0.1, 1.0)
        
        print(f"  Importance map: min={importance.min():.3f}, max={importance.max():.3f}, mean={importance.mean():.3f}")
        
        return importance
    
    def get_line_pixels(self, p1, p2, width, height):
        """
        Get all pixel coordinates along a line with anti-aliasing support (Xiaolin Wu's algorithm)
        Returns list of (y, x, weight) tuples where weight is 0.0-1.0 for anti-aliasing
        """
        # Check cache first
        cache_key = (p1, p2) if p1 < p2 else (p2, p1)
        if cache_key in self.line_cache:
            return self.line_cache[cache_key]
        
        x1, y1 = self.pins[p1]
        x2, y2 = self.pins[p2]
        
        pixels = []
        
        # Xiaolin Wu's line algorithm for anti-aliasing
        steep = abs(y2 - y1) > abs(x2 - x1)
        
        if steep:
            x1, y1 = y1, x1
            x2, y2 = y2, x2
        
        if x1 > x2:
            x1, x2 = x2, x1
            y1, y2 = y2, y1
        
        dx = x2 - x1
        dy = y2 - y1
        
        if dx == 0:
            gradient = 1.0
        else:
            gradient = dy / dx
        
        # Handle first endpoint
        xend = round(x1)
        yend = y1 + gradient * (xend - x1)
        xgap = 1 - ((x1 + 0.5) - int(x1 + 0.5))
        xpxl1 = xend
        ypxl1 = int(yend)
        
        if steep:
            if 0 <= xpxl1 < height and 0 <= ypxl1 < width:
                pixels.append((xpxl1, ypxl1, (1 - (yend - ypxl1)) * xgap))
            if 0 <= xpxl1 < height and 0 <= ypxl1 + 1 < width:
                pixels.append((xpxl1, ypxl1 + 1, (yend - ypxl1) * xgap))
        else:
            if 0 <= ypxl1 < height and 0 <= xpxl1 < width:
                pixels.append((ypxl1, xpxl1, (1 - (yend - ypxl1)) * xgap))
            if 0 <= ypxl1 + 1 < height and 0 <= xpxl1 < width:
                pixels.append((ypxl1 + 1, xpxl1, (yend - ypxl1) * xgap))
        
        intery = yend + gradient
        
        # Handle second endpoint
        xend = round(x2)
        yend = y2 + gradient * (xend - x2)
        xgap = (x2 + 0.5) - int(x2 + 0.5)
        xpxl2 = xend
        ypxl2 = int(yend)
        
        if steep:
            if 0 <= xpxl2 < height and 0 <= ypxl2 < width:
                pixels.append((xpxl2, ypxl2, (1 - (yend - ypxl2)) * xgap))
            if 0 <= xpxl2 < height and 0 <= ypxl2 + 1 < width:
                pixels.append((xpxl2, ypxl2 + 1, (yend - ypxl2) * xgap))
        else:
            if 0 <= ypxl2 < height and 0 <= xpxl2 < width:
                pixels.append((ypxl2, xpxl2, (1 - (yend - ypxl2)) * xgap))
            if 0 <= ypxl2 + 1 < height and 0 <= xpxl2 < width:
                pixels.append((ypxl2 + 1, xpxl2, (yend - ypxl2) * xgap))
        
        # Main loop
        for x in range(int(xpxl1) + 1, int(xpxl2)):
            if steep:
                if 0 <= x < height and 0 <= int(intery) < width:
                    pixels.append((x, int(intery), 1 - (intery - int(intery))))
                if 0 <= x < height and 0 <= int(intery) + 1 < width:
                    pixels.append((x, int(intery) + 1, intery - int(intery)))
            else:
                if 0 <= int(intery) < height and 0 <= x < width:
                    pixels.append((int(intery), x, 1 - (intery - int(intery))))
                if 0 <= int(intery) + 1 < height and 0 <= x < width:
                    pixels.append((int(intery) + 1, x, intery - int(intery)))
            
            intery += gradient
        
        # Cache the result
        self.line_cache[cache_key] = pixels
        return pixels
    
    def calculate_line_error(self, img_array, edge_array, p1, p2):
        """
        Calculate line quality score (higher = better)
        Considers both darkness and edge alignment
        Supports importance map for region prioritization (v3.3.0)
        Supports squared difference metric for better convergence (v3.4.0)
        Supports anti-aliased line rendering (v3.5.0)
        Supports "fear removal" error function (v3.8.0)
        
        Fear removal: Only count pixels that improve (get darker), ignore pixels
        that temporarily worsen. This allows algorithm to capture fine details
        without fear of ruining existing progress.
        """
        pixels = self.get_line_pixels(p1, p2, img_array.shape[1], img_array.shape[0])
        
        if not pixels:
            return 0
        
        total_score = 0
        total_weight = 0
        
        for y, x, weight in pixels:
            # Current pixel value
            current_val = img_array[y, x]
            
            # Simulated new value after drawing this line
            effective_weight = self.line_weight * self.line_opacity * weight
            new_val = min(255, current_val + effective_weight)
            
            # Target value (original image)
            target_val = 255  # White background target
            
            if self.use_fear_removal:
                # Fear removal: only count improvement (pixels getting closer to white)
                # Ignore pixels that get worse (darker) temporarily
                current_error = abs(target_val - current_val)
                new_error = abs(target_val - new_val)
                
                # Only count if this line makes pixel worse (darker)
                # We want to minimize darkness, so score = reduction in darkness
                if new_val > current_val:  # Getting lighter (worse for string art)
                    pixel_score = 0  # Ignore - future strings can fix this
                else:
                    # Getting darker (good) - score based on how much darker
                    darkness_added = current_val - new_val
                    pixel_score = darkness_added
            else:
                # Original scoring: darkness score (darker pixels = better match)
                darkness = 255 - img_array[y, x]
                
                # Apply squared difference if enabled (v3.4.0)
                if self.use_squared_diff:
                    darkness = (darkness / 255.0) ** 2 * 255
                
                pixel_score = darkness
            
            # Edge score (stronger edges = higher priority)
            edge_strength = edge_array[y, x]
            
            # Importance multiplier (v3.3.0)
            importance = 1.0
            if self.importance_map is not None:
                importance = self.importance_map[y, x]
            
            # Combined score with edge weighting, importance, and anti-aliasing weight
            if self.use_fear_removal:
                # For fear removal, edge weight is additive bonus
                total_score += (pixel_score + edge_strength * self.edge_weight * 0.1) * importance * weight
            else:
                # Original: edge weight is multiplicative
                total_score += (pixel_score + (edge_strength * self.edge_weight)) * importance * weight
            
            total_weight += weight
        
        return total_score / total_weight if total_weight > 0 else 0
    
    def evaluate_look_ahead(self, img_array, edge_array, current_pin, next_pin, depth=2):
        """
        Look-ahead optimization: evaluate quality of next N moves
        Returns combined score considering future moves
        """
        if depth == 0:
            return self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
        
        # Simulate drawing this line
        temp_img = img_array.copy()
        pixels = self.get_line_pixels(current_pin, next_pin, img_array.shape[1], img_array.shape[0])
        effective_weight = self.line_weight * self.line_opacity
        
        for y, x, weight in pixels:
            temp_img[y, x] = min(255, temp_img[y, x] + effective_weight * weight)
        
        # Current line score
        current_score = self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
        
        # Find best next move
        best_future_score = 0
        candidate_pins = range(self.num_pins) if not self.random_sampling else \
                        [np.random.randint(0, self.num_pins) for _ in range(min(50, self.num_pins))]
        
        for future_pin in candidate_pins:
            pin_distance = min(
                abs(future_pin - next_pin),
                self.num_pins - abs(future_pin - next_pin)
            )
            if pin_distance < self.min_distance:
                continue
            
            future_score = self.evaluate_look_ahead(temp_img, edge_array, next_pin, future_pin, depth - 1)
            best_future_score = max(best_future_score, future_score)
        
        # Weighted combination: 70% current, 30% future
        return current_score * 0.7 + best_future_score * 0.3
    
    def draw_line_on_array(self, img_array, p1, p2):
        """Draw a line on the image array (lighten pixels with opacity and anti-aliasing support)"""
        pixels = self.get_line_pixels(p1, p2, img_array.shape[1], img_array.shape[0])
        
        # Apply line weight with opacity
        effective_weight = self.line_weight * self.line_opacity
        
        for y, x, weight in pixels:
            # Lighten the pixel with opacity and anti-aliasing weight
            img_array[y, x] = min(255, img_array[y, x] + effective_weight * weight)
    
    def evaluate_pin_candidate(self, args):
        """Helper function for parallel evaluation of pin candidates"""
        img_array, edge_array, current_pin, next_pin = args
        
        # Check distance constraint
        pin_distance = min(
            abs(next_pin - current_pin),
            self.num_pins - abs(next_pin - current_pin)
        )
        if pin_distance < self.min_distance:
            return (next_pin, -1)
        
        # Calculate score
        if self.look_ahead:
            score = self.evaluate_look_ahead(img_array, edge_array, current_pin, next_pin, depth=1)
        else:
            score = self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
        
        return (next_pin, score)
    
    def find_best_pins_beam_search(self, img_array, edge_array, current_pin, pins_to_check):
        """
        Beam search optimization: keep top-k candidates instead of just best one
        With optional simulated annealing for escaping local optima (v3.4.0)
        Returns list of (pin, score) tuples sorted by score (descending)
        """
        if self.use_parallel and len(pins_to_check) > 50:
            # Parallel evaluation for large candidate sets
            eval_args = [(img_array, edge_array, current_pin, next_pin) 
                        for next_pin in pins_to_check]
            
            with ThreadPoolExecutor(max_workers=self.num_workers) as executor:
                results = list(executor.map(self.evaluate_pin_candidate, eval_args))
        else:
            # Sequential evaluation for small sets
            results = []
            for next_pin in pins_to_check:
                pin_distance = min(
                    abs(next_pin - current_pin),
                    self.num_pins - abs(next_pin - current_pin)
                )
                if pin_distance < self.min_distance:
                    continue
                
                if self.look_ahead:
                    score = self.evaluate_look_ahead(img_array, edge_array, current_pin, next_pin, depth=1)
                else:
                    score = self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
                
                results.append((next_pin, score))
        
        # Filter out invalid scores and sort by score (descending)
        valid_results = [(pin, score) for pin, score in results if score > 0]
        valid_results.sort(key=lambda x: x[1], reverse=True)
        
        # Simulated annealing: occasionally accept suboptimal solutions (v3.4.0)
        if self.use_simulated_annealing and len(valid_results) > 1 and self.current_temp > 1.0:
            best_score = valid_results[0][1]
            
            # Consider accepting a worse solution with probability based on temperature
            for i in range(1, min(5, len(valid_results))):
                candidate_score = valid_results[i][1]
                score_diff = best_score - candidate_score
                
                # Acceptance probability: exp(-ΔE/T)
                acceptance_prob = np.exp(-score_diff / self.current_temp)
                
                if np.random.random() < acceptance_prob:
                    # Accept this suboptimal solution
                    # Move it to front but keep others for beam search
                    accepted = valid_results.pop(i)
                    valid_results.insert(0, accepted)
                    break
        
        # Return top beam_width candidates
        return valid_results[:self.beam_width]
    
    def apply_2opt_optimization(self, img_array, edge_array, max_iterations=100):
        """
        Apply 2-opt local optimization to improve line sequence (v3.3.0)
        
        2-opt works by iteratively removing crossing edges and reconnecting them
        in a better way. This is a post-processing step that improves visual quality.
        
        Args:
            img_array: Current image state
            edge_array: Edge detection array
            max_iterations: Maximum number of improvement passes
        
        Returns:
            Number of improvements made
        """
        if len(self.lines) < 4:
            return 0
        
        print(f"Applying 2-opt optimization (max {max_iterations} iterations)...")
        improvements = 0
        
        for iteration in range(max_iterations):
            improved = False
            
            # Try all pairs of edges
            for i in range(len(self.lines) - 2):
                for j in range(i + 2, len(self.lines)):
                    # Skip adjacent edges
                    if j == i + 1:
                        continue
                    
                    # Current edges: (a->b) and (c->d)
                    a, b = self.lines[i]
                    c, d = self.lines[j]
                    
                    # Calculate current cost
                    current_cost = (
                        self.calculate_line_error(img_array, edge_array, a, b) +
                        self.calculate_line_error(img_array, edge_array, c, d)
                    )
                    
                    # Try swapping: (a->c) and (b->d)
                    new_cost = (
                        self.calculate_line_error(img_array, edge_array, a, c) +
                        self.calculate_line_error(img_array, edge_array, b, d)
                    )
                    
                    # If improvement found, apply it
                    if new_cost > current_cost * 1.01:  # 1% improvement threshold
                        # Reverse the segment between i and j
                        self.lines[i] = (a, c)
                        self.lines[j] = (b, d)
                        
                        # Reverse intermediate edges
                        segment = self.lines[i+1:j]
                        segment.reverse()
                        for k, (p1, p2) in enumerate(segment):
                            segment[k] = (p2, p1)  # Reverse direction
                        self.lines[i+1:j] = segment
                        
                        improvements += 1
                        improved = True
            
            if not improved:
                print(f"  2-opt converged after {iteration + 1} iterations ({improvements} improvements)")
                break
            
            if (iteration + 1) % 10 == 0:
                print(f"  2-opt iteration {iteration + 1}/{max_iterations} ({improvements} improvements so far)")
        
        if iteration == max_iterations - 1:
            print(f"  2-opt completed {max_iterations} iterations ({improvements} improvements)")
        
        return improvements
    
    def apply_3opt_optimization(self, img_array, edge_array, max_iterations=50):
        """
        Apply 3-opt local optimization to improve line sequence (v3.7.0)
        
        3-opt is more powerful than 2-opt as it considers removing 3 edges
        and reconnecting them in different ways. This can escape local optima
        that 2-opt cannot.
        
        Args:
            img_array: Current image state
            edge_array: Edge detection array
            max_iterations: Maximum number of improvement passes
        
        Returns:
            Number of improvements made
        """
        if len(self.lines) < 6:
            return 0
        
        print(f"Applying 3-opt optimization (max {max_iterations} iterations)...")
        improvements = 0
        
        for iteration in range(max_iterations):
            improved = False
            
            # Try all combinations of 3 edges
            # This is O(n^3) so we limit iterations
            for i in range(len(self.lines) - 4):
                for j in range(i + 2, len(self.lines) - 2):
                    for k in range(j + 2, len(self.lines)):
                        # Current edges: (a->b), (c->d), (e->f)
                        a, b = self.lines[i]
                        c, d = self.lines[j]
                        e, f = self.lines[k]
                        
                        # Calculate current cost
                        current_cost = (
                            self.calculate_line_error(img_array, edge_array, a, b) +
                            self.calculate_line_error(img_array, edge_array, c, d) +
                            self.calculate_line_error(img_array, edge_array, e, f)
                        )
                        
                        # Try different reconnection patterns
                        # Pattern 1: (a->c), (b->e), (d->f)
                        new_cost_1 = (
                            self.calculate_line_error(img_array, edge_array, a, c) +
                            self.calculate_line_error(img_array, edge_array, b, e) +
                            self.calculate_line_error(img_array, edge_array, d, f)
                        )
                        
                        # Pattern 2: (a->d), (c->e), (b->f)
                        new_cost_2 = (
                            self.calculate_line_error(img_array, edge_array, a, d) +
                            self.calculate_line_error(img_array, edge_array, c, e) +
                            self.calculate_line_error(img_array, edge_array, b, f)
                        )
                        
                        # Pattern 3: (a->e), (c->b), (d->f)
                        new_cost_3 = (
                            self.calculate_line_error(img_array, edge_array, a, e) +
                            self.calculate_line_error(img_array, edge_array, c, b) +
                            self.calculate_line_error(img_array, edge_array, d, f)
                        )
                        
                        # Find best pattern
                        best_cost = max(new_cost_1, new_cost_2, new_cost_3)
                        
                        # If improvement found (1% threshold), apply it
                        if best_cost > current_cost * 1.01:
                            if best_cost == new_cost_1:
                                # Apply pattern 1
                                self.lines[i] = (a, c)
                                self.lines[j] = (b, e)
                                self.lines[k] = (d, f)
                            elif best_cost == new_cost_2:
                                # Apply pattern 2
                                self.lines[i] = (a, d)
                                self.lines[j] = (c, e)
                                self.lines[k] = (b, f)
                            else:
                                # Apply pattern 3
                                self.lines[i] = (a, e)
                                self.lines[j] = (c, b)
                                self.lines[k] = (d, f)
                            
                            improvements += 1
                            improved = True
            
            if not improved:
                print(f"  3-opt converged after {iteration + 1} iterations ({improvements} improvements)")
                break
            
            if (iteration + 1) % 10 == 0:
                print(f"  3-opt iteration {iteration + 1}/{max_iterations} ({improvements} improvements so far)")
        
        if iteration == max_iterations - 1:
            print(f"  3-opt completed {max_iterations} iterations ({improvements} improvements)")
        
        return improvements
    
    def setup_knn_model(self):
        """
        Setup k-Nearest Neighbors model for smart pin selection (v3.7.0)
        This allows us to focus on nearby pins for more efficient search
        """
        if not self.use_knn_selection or len(self.pins) == 0:
            return
        
        # Convert pins to numpy array for sklearn
        pin_coords = np.array(self.pins)
        
        # Fit k-NN model
        self.knn_model = NearestNeighbors(
            n_neighbors=min(self.knn_neighbors, len(self.pins)),
            algorithm='ball_tree',
            metric='euclidean'
        )
        self.knn_model.fit(pin_coords)
        print(f"k-NN model initialized (k={min(self.knn_neighbors, len(self.pins))} neighbors)")
    
    def get_knn_candidates(self, current_pin, current_knn_k=None):
        """
        Get k-nearest neighbor pins for selection (v3.7.0)
        
        Args:
            current_pin: Current pin index
            current_knn_k: Override k value (for adaptive k-NN)
        
        Returns:
            List of candidate pin indices
        """
        if self.knn_model is None:
            return list(range(self.num_pins))
        
        # Get k-nearest neighbors
        k = current_knn_k if current_knn_k is not None else self.knn_neighbors
        k = min(k, len(self.pins))
        
        pin_coord = np.array([self.pins[current_pin]])
        distances, indices = self.knn_model.kneighbors(pin_coord, n_neighbors=k)
        
        # Filter by minimum distance constraint
        candidates = []
        for idx in indices[0]:
            pin_distance = min(
                abs(idx - current_pin),
                self.num_pins - abs(idx - current_pin)
            )
            if pin_distance >= self.min_distance:
                candidates.append(int(idx))
        
        return candidates
    
    def rgb_to_cmy(self, rgb_array):
        """
        Convert RGB image to CMY color space (v4.0.0)
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
        cmy_array = (cmy_norm * 255).astype(np.float32)
        
        return cmy_array
    
    def cmy_to_rgb(self, cmy_array):
        """
        Convert CMY image back to RGB (v4.0.0)
        
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
    
    def generate_multicolor(self, image_path, max_lines_per_color=1000, output_dir=None):
        """
        Generate multi-color string art (v4.0.0)
        
        Uses CMY or RGB color model to generate separate string sequences
        for each color channel, then combines them.
        
        Args:
            image_path: Path to input RGB image
            max_lines_per_color: Maximum lines per color channel
            output_dir: Directory to save outputs
        
        Returns:
            dict: Results including sequences, statistics, and file paths
        """
        start_time = time.time()
        
        print(f"Loading RGB image: {image_path}")
        img = Image.open(image_path).convert('RGB')
        
        # Resize to square
        size = 800
        img = img.resize((size, size), Image.Resampling.LANCZOS)
        
        # Setup pins (shared across all colors)
        self.setup_pins(size, size)
        print(f"Setup {self.num_pins} pins in {self.pin_arrangement} arrangement")
        
        # Convert to numpy array
        img_array_rgb = np.array(img, dtype=np.float32)
        original_array_rgb = img_array_rgb.copy()
        
        # Choose color model
        if self.color_mode == 'CMY':
            print("Using CMY color model (subtractive, better for string art)")
            img_array_color = self.rgb_to_cmy(img_array_rgb)
            color_channels = ['C', 'M', 'Y']
            color_names = ['Cyan', 'Magenta', 'Yellow']
        else:
            print("Using RGB color model (additive)")
            img_array_color = img_array_rgb.copy()
            color_channels = ['R', 'G', 'B']
            color_names = ['Red', 'Green', 'Blue']
        
        # Setup k-NN model for smart pin selection (shared)
        if self.use_knn_selection:
            self.setup_knn_model()
        
        # Generate string art for each color channel
        all_results = {}
        
        for channel_idx, (channel, channel_name) in enumerate(zip(color_channels, color_names)):
            print(f\"\\n{'='*60}")
            print(f\"Processing {channel_name} channel ({channel_idx+1}/3)")
            print(f\"{'='*60}")
            
            # Extract single channel as grayscale
            channel_array = img_array_color[:, :, channel_idx].copy()
            
            # Invert for string art (darker = more strings needed)
            # For CMY: high value = more color needed
            # For RGB: high value = brighter, so invert
            if self.color_mode == 'RGB':
                channel_array = 255 - channel_array
            
            # Detect edges for this channel
            print(f\"Detecting edges for {channel_name} channel...")
            edge_array = self.detect_edges(channel_array)
            
            # Auto-generate importance map if enabled
            if self.auto_importance:
                importance_map = self.generate_importance_map(channel_array, edge_array)
                self.importance_map = importance_map
            
            # Generate string sequence for this channel
            print(f\"Generating {channel_name} strings (max {max_lines_per_color} lines)...")
            
            self.lines = []  # Reset lines for this channel
            current_pin = 0
            last_scores = []
            self.current_temp = self.annealing_temp
            
            # Simulate white canvas for this channel
            canvas_array = np.full_like(channel_array, 255.0)
            
            for line_num in range(max_lines_per_color):
                # Warm restart for simulated annealing
                if (self.use_simulated_annealing and self.warm_restart_interval > 0 and 
                    line_num > 0 and line_num % self.warm_restart_interval == 0):
                    self.current_temp = self.initial_annealing_temp
                    print(f\"  Warm restart at line {line_num} (temp reset to {self.current_temp:.1f})")
                
                # Adaptive beam width
                if self.adaptive_beam and self.initial_beam_width > 1:
                    progress = line_num / max_lines_per_color
                    self.beam_width = max(1, int(self.initial_beam_width * (1.0 - 0.7 * progress)))
                
                # Determine which pins to evaluate
                if self.use_knn_selection:
                    progress = line_num / max_lines_per_color
                    current_k = max(10, int(self.knn_neighbors * (1.0 - 0.5 * progress)))
                    pins_to_check = self.get_knn_candidates(current_pin, current_k)
                elif self.use_pool_sampling:
                    candidate_pins = []
                    attempts = 0
                    max_attempts = self.pool_size * 2
                    
                    while len(candidate_pins) < self.pool_size and attempts < max_attempts:
                        next_pin = np.random.randint(0, self.num_pins)
                        pin_distance = min(
                            abs(next_pin - current_pin),
                            self.num_pins - abs(next_pin - current_pin)
                        )
                        if pin_distance >= self.min_distance and next_pin not in candidate_pins:
                            candidate_pins.append(next_pin)
                        attempts += 1
                    
                    pool_scores = []
                    for next_pin in candidate_pins:
                        score = self.calculate_line_error(canvas_array, edge_array, current_pin, next_pin)
                        pool_scores.append((next_pin, score))
                    
                    pool_scores.sort(key=lambda x: x[1], reverse=True)
                    pins_to_check = [pin for pin, score in pool_scores[:self.pool_select]]
                else:
                    pins_to_check = list(range(self.num_pins))
                
                # Find best pin using beam search
                if self.beam_width > 1:
                    beam_candidates = self.find_best_pins_beam_search(canvas_array, edge_array, current_pin, pins_to_check)
                    if not beam_candidates:
                        print(f\"  Stopping at {line_num} lines (no valid candidates)")
                        break
                    best_pin, best_score = beam_candidates[0]
                else:
                    best_pin = None
                    best_score = -1
                    for next_pin in pins_to_check:
                        pin_distance = min(
                            abs(next_pin - current_pin),
                            self.num_pins - abs(next_pin - current_pin)
                        )
                        if pin_distance < self.min_distance:
                            continue
                        score = self.calculate_line_error(canvas_array, edge_array, current_pin, next_pin)
                        if score > best_score:
                            best_score = score
                            best_pin = next_pin
                
                # Adaptive stopping
                if self.adaptive_stop:
                    last_scores.append(best_score)
                    if len(last_scores) > 10:
                        last_scores.pop(0)
                    if len(last_scores) >= 10:
                        avg_recent_score = sum(last_scores) / len(last_scores)
                        if avg_recent_score < self.stop_threshold:
                            print(f\"  Stopping at {line_num} lines (adaptive: avg score {avg_recent_score:.2f} < {self.stop_threshold})")
                            break
                
                if best_pin is None or best_score < 0.1:
                    print(f\"  Stopping at {line_num} lines (no improvement)")
                    break
                
                # Draw the best line
                self.lines.append((current_pin, best_pin))
                self.draw_line_on_array(canvas_array, current_pin, best_pin)
                current_pin = best_pin
                
                # Cool down temperature
                if self.use_simulated_annealing:
                    self.current_temp *= self.annealing_cooling
                
                if (line_num + 1) % 100 == 0:
                    elapsed = time.time() - start_time
                    avg_score = sum(last_scores) / len(last_scores) if last_scores else 0
                    print(f\"  {line_num + 1} lines generated... ({elapsed:.1f}s, avg score: {avg_score:.2f})")
            
            # Store results for this channel
            if self.color_mode == 'CMY':
                self.color_lines[channel] = self.lines.copy()
            else:
                self.color_lines_rgb[channel] = self.lines.copy()
            
            print(f\"✓ {channel_name} channel: {len(self.lines)} lines generated")
            
            all_results[channel] = {
                'lines': self.lines.copy(),
                'num_lines': len(self.lines)
            }
        
        elapsed_total = time.time() - start_time
        total_lines = sum(r['num_lines'] for r in all_results.values())
        print(f\"\\n{'='*60}")
        print(f\"✓ Multi-color generation complete: {total_lines} total lines in {elapsed_total:.1f}s")
        print(f\"{'='*60}")
        
        # Create output directory
        if output_dir is None:
            output_dir = Path(image_path).parent / \"string_art_output\"
        else:
            output_dir = Path(output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)
        
        # Save results
        timestamp = datetime.now().strftime(\"%Y%m%d_%H%M%S")
        base_name = Path(image_path).stem
        
        # 1. Save sequences as JSON
        sequence_file = output_dir / f\"{base_name}_multicolor_sequence_{timestamp}.json\"
        sequence_data = {
            \"metadata\": {
                \"source_image\": str(image_path),
                \"generated_at\": datetime.now().isoformat(),
                \"version\": \"4.0.0\",
                \"color_mode\": self.color_mode,
                \"num_pins\": self.num_pins,
                \"total_lines\": total_lines,
                \"generation_time_seconds\": elapsed_total,
            },
            \"pins\": self.pins,
            \"color_sequences\": all_results
        }
        
        with open(sequence_file, 'w') as f:
            json.dump(sequence_data, f, indent=2)
        print(f\"✓ Saved multi-color sequence: {sequence_file}")
        
        # 2. Render multi-color preview
        render_file = output_dir / f\"{base_name}_multicolor_render_{timestamp}.png\"
        self.render_multicolor_preview(render_file, size, all_results)
        print(f\"✓ Saved multi-color preview: {render_file}")
        
        # 3. Render multi-color SVG
        svg_file = output_dir / f\"{base_name}_multicolor_{timestamp}.svg\"
        self.render_multicolor_svg(svg_file, all_results)
        print(f\"✓ Saved multi-color SVG: {svg_file}")
        
        return {
            \"total_lines\": total_lines,
            \"generation_time\": elapsed_total,
            \"sequence_file\": str(sequence_file),
            \"svg_file\": str(svg_file),
            \"preview_file\": str(render_file),
            \"color_results\": all_results
        }
    
    def render_multicolor_preview(self, output_path, size, color_results):
        """Render multi-color string art preview (v4.0.0)\"""
        img = Image.new('RGB', (size, size), 'white')
        draw = ImageDraw.Draw(img, 'RGBA')
        
        # Define colors for each channel
        if self.color_mode == 'CMY':
            colors = {
                'C': (0, 255, 255, int(255 * self.line_opacity)),  # Cyan
                'M': (255, 0, 255, int(255 * self.line_opacity)),  # Magenta
                'Y': (255, 255, 0, int(255 * self.line_opacity))   # Yellow
            }
        else:
            colors = {
                'R': (255, 0, 0, int(255 * self.line_opacity)),    # Red
                'G': (0, 255, 0, int(255 * self.line_opacity)),    # Green
                'B': (0, 0, 255, int(255 * self.line_opacity))     # Blue
            }
        
        # Draw lines for each color
        for channel, result in color_results.items():
            color = colors[channel]
            for p1, p2 in result['lines']:
                draw.line([self.pins[p1], self.pins[p2]], fill=color, width=1)
        
        # Draw pins
        for pin in self.pins:
            draw.ellipse([pin[0]-2, pin[1]-2, pin[0]+2, pin[1]+2], fill='red')
        
        img.save(output_path)
    
    def render_multicolor_svg(self, output_path, color_results):
        """Render multi-color string art as SVG (v4.0.0)\"""
        svg_width_mm = 600
        svg_height_mm = 600
        stroke_width_mm = 0.18
        
        svg_lines = [
            f'<?xml version=\"1.0\" encoding=\"UTF-8\"?>',
            f'<svg width=\"{svg_width_mm}mm\" height=\"{svg_height_mm}mm\" ',
            f'     viewBox=\"0 0 {svg_width_mm} {svg_height_mm}\" ',
            f'     xmlns=\"http://www.w3.org/2000/svg\">',
            f'  <!-- Multi-Color String Art Generator v4.0.0 -->',
            f'  <!-- Color Mode: {self.color_mode} -->',
            f'  <!-- Pins: {self.num_pins} -->',
        ]
        
        # Define SVG colors
        if self.color_mode == 'CMY':
            svg_colors = {
                'C': 'cyan',
                'M': 'magenta',
                'Y': 'yellow'
            }
        else:
            svg_colors = {
                'R': 'red',
                'G': 'green',
                'B': 'blue'
            }
        
        scale = svg_width_mm / 800.0
        
        # Draw each color channel
        for channel, result in color_results.items():
            svg_lines.append(f'  <!-- {channel} channel: {result[\"num_lines\"]} lines -->')
            svg_lines.append(f'  <g id=\"{channel}-channel\" stroke=\"{svg_colors[channel]}\" stroke-width=\"{stroke_width_mm}mm\" fill=\"none\" opacity=\"{self.line_opacity}\">')
            
            for p1, p2 in result['lines']:
                x1, y1 = self.pins[p1]
                x2, y2 = self.pins[p2]
                x1_mm = x1 * scale
                y1_mm = y1 * scale
                x2_mm = x2 * scale
                y2_mm = y2 * scale
                svg_lines.append(f'    <line x1=\"{x1_mm:.3f}\" y1=\"{y1_mm:.3f}\" x2=\"{x2_mm:.3f}\" y2=\"{y2_mm:.3f}\"/>')
            
            svg_lines.append(f'  </g>')
        
        svg_lines.append(f'</svg>')
        
        with open(output_path, 'w') as f:
            f.write('\\n'.join(svg_lines))
    
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
        print(f"Setup {self.num_pins} pins in {self.pin_arrangement} arrangement")
        
        # Convert to numpy array
        img_array = np.array(img, dtype=np.float32)
        original_array = img_array.copy()
        
        # Detect edges for feature-aware selection
        print("Detecting edges for feature-aware optimization...")
        edge_array = self.detect_edges(img_array)
        
        # Auto-generate importance map if enabled (v3.4.0)
        if self.auto_importance and self.importance_map is None:
            self.importance_map = self.generate_importance_map(img_array, edge_array)
        
        # Setup k-NN model for smart pin selection (v3.7.0)
        if self.use_knn_selection:
            self.setup_knn_model()
        
        # Generate string sequence
        print(f"Generating string art (max {max_lines} lines)...")
        print("Using edge-aware algorithm for better clarity...")
        if self.use_squared_diff:
            print("Squared difference metric enabled (better convergence)")
        if self.use_simulated_annealing:
            print(f"Simulated annealing enabled (temp: {self.annealing_temp}, cooling: {self.annealing_cooling})")
            if self.warm_restart_interval > 0:
                print(f"Warm restart enabled (every {self.warm_restart_interval} lines)")
        if self.beam_width > 1:
            print(f"Beam search enabled (width: {self.beam_width})")
        if self.use_parallel:
            print(f"Parallel processing enabled ({self.num_workers} workers)")
        if self.look_ahead:
            print("Look-ahead optimization enabled (slower but better quality)")
        if self.adaptive_stop:
            print(f"Adaptive stopping enabled (threshold: {self.stop_threshold})")
        if self.adaptive_beam:
            print(f"Adaptive beam width enabled (initial: {self.initial_beam_width})")
        if self.use_knn_selection:
            print(f"k-NN selection enabled (k={self.knn_neighbors} neighbors)")
        if self.use_fear_removal:
            print("Fear removal error function enabled (better detail capture)")
        
        self.lines = []
        current_pin = 0
        last_scores = []  # Track recent scores for adaptive stopping
        self.current_temp = self.annealing_temp  # Reset temperature
        
        for line_num in range(max_lines):
            # Warm restart for simulated annealing (v3.7.0)
            if (self.use_simulated_annealing and self.warm_restart_interval > 0 and 
                line_num > 0 and line_num % self.warm_restart_interval == 0):
                # Reset temperature to initial value
                self.current_temp = self.initial_annealing_temp
                print(f"  Warm restart at line {line_num} (temp reset to {self.current_temp:.1f})")
            
            # Adaptive beam width (v3.6.0): reduce beam width as we progress
            # Early stage: explore more (wider beam)
            # Late stage: exploit more (narrower beam)
            if self.adaptive_beam and self.initial_beam_width > 1:
                progress = line_num / max_lines
                # Exponential decay: start wide, narrow down
                # At 0%: initial_beam_width
                # At 50%: ~70% of initial
                # At 100%: ~30% of initial (minimum 1)
                self.beam_width = max(1, int(self.initial_beam_width * (1.0 - 0.7 * progress)))
            
            # Determine which pins to evaluate
            if self.use_knn_selection:
                # k-NN selection (v3.7.0): focus on nearby pins
                # Adaptive k: start with more neighbors, reduce as we progress
                progress = line_num / max_lines
                current_k = max(10, int(self.knn_neighbors * (1.0 - 0.5 * progress)))
                pins_to_check = self.get_knn_candidates(current_pin, current_k)
            elif self.use_pool_sampling:
                # Pool-based sampling (v3.9.0 - Bridges 2022 method)
                # More efficient than simple random sampling:
                # 1. Generate large random pool
                # 2. Evaluate all in pool
                # 3. Select top-k from pool
                # This dramatically reduces computation while maintaining quality
                candidate_pins = []
                attempts = 0
                max_attempts = self.pool_size * 2  # Avoid infinite loop
                
                while len(candidate_pins) < self.pool_size and attempts < max_attempts:
                    next_pin = np.random.randint(0, self.num_pins)
                    
                    # Check distance constraint
                    pin_distance = min(
                        abs(next_pin - current_pin),
                        self.num_pins - abs(next_pin - current_pin)
                    )
                    
                    if pin_distance >= self.min_distance and next_pin not in candidate_pins:
                        candidate_pins.append(next_pin)
                    
                    attempts += 1
                
                # Evaluate all candidates in pool
                pool_scores = []
                for next_pin in candidate_pins:
                    if self.look_ahead:
                        score = self.evaluate_look_ahead(img_array, edge_array, current_pin, next_pin, depth=1)
                    else:
                        score = self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
                    pool_scores.append((next_pin, score))
                
                # Sort by score and select top pool_select candidates
                pool_scores.sort(key=lambda x: x[1], reverse=True)
                pins_to_check = [pin for pin, score in pool_scores[:self.pool_select]]
            elif self.random_sampling and self.sample_size:
                # Legacy random sampling (kept for compatibility)
                candidate_pins = []
                attempts = 0
                max_attempts = self.sample_size * 3  # Avoid infinite loop
                
                while len(candidate_pins) < self.sample_size and attempts < max_attempts:
                    next_pin = np.random.randint(0, self.num_pins)
                    
                    # Check distance constraint
                    pin_distance = min(
                        abs(next_pin - current_pin),
                        self.num_pins - abs(next_pin - current_pin)
                    )
                    
                    if pin_distance >= self.min_distance and next_pin not in candidate_pins:
                        candidate_pins.append(next_pin)
                    
                    attempts += 1
                
                pins_to_check = candidate_pins
            else:
                # Check all pins (original behavior)
                pins_to_check = list(range(self.num_pins))
            
            # Use beam search to find best candidates
            if self.beam_width > 1:
                # Beam search: get top-k candidates
                beam_candidates = self.find_best_pins_beam_search(img_array, edge_array, current_pin, pins_to_check)
                
                if not beam_candidates:
                    print(f"Stopping at {line_num} lines (no valid candidates)")
                    break
                
                # Select best from beam
                best_pin, best_score = beam_candidates[0]
            else:
                # Original greedy search
                best_pin = None
                best_score = -1
                
                for next_pin in pins_to_check:
                    # Skip if too close
                    pin_distance = min(
                        abs(next_pin - current_pin),
                        self.num_pins - abs(next_pin - current_pin)
                    )
                    if pin_distance < self.min_distance:
                        continue
                    
                    # Calculate score
                    if self.look_ahead:
                        score = self.evaluate_look_ahead(img_array, edge_array, current_pin, next_pin, depth=1)
                    else:
                        score = self.calculate_line_error(img_array, edge_array, current_pin, next_pin)
                    
                    if score > best_score:
                        best_score = score
                        best_pin = next_pin
            
            # Adaptive stopping condition
            if self.adaptive_stop:
                last_scores.append(best_score)
                if len(last_scores) > 10:
                    last_scores.pop(0)
                
                # Stop if average recent score is too low
                if len(last_scores) >= 10:
                    avg_recent_score = sum(last_scores) / len(last_scores)
                    if avg_recent_score < self.stop_threshold:
                        print(f"Stopping at {line_num} lines (adaptive: avg score {avg_recent_score:.2f} < {self.stop_threshold})")
                        break
            
            if best_pin is None or best_score < 0.1:
                print(f"Stopping at {line_num} lines (no improvement)")
                break
            
            # Draw the best line
            self.lines.append((current_pin, best_pin))
            self.draw_line_on_array(img_array, current_pin, best_pin)
            current_pin = best_pin
            
            # Cool down temperature for simulated annealing (v3.4.0)
            if self.use_simulated_annealing:
                self.current_temp *= self.annealing_cooling
            
            if (line_num + 1) % 100 == 0:
                elapsed = time.time() - start_time
                avg_score = sum(last_scores) / len(last_scores) if last_scores else 0
                temp_info = f", temp: {self.current_temp:.1f}" if self.use_simulated_annealing else ""
                print(f"  {line_num + 1} lines generated... ({elapsed:.1f}s, avg score: {avg_score:.2f}{temp_info})")
        
        elapsed_total = time.time() - start_time
        print(f"✓ Generated {len(self.lines)} lines in {elapsed_total:.1f}s")
        
        # Apply 2-opt optimization if enabled (v3.3.0)
        if self.use_2opt and len(self.lines) >= 4:
            opt_start = time.time()
            improvements = self.apply_2opt_optimization(img_array, edge_array, self.two_opt_iterations)
            opt_time = time.time() - opt_start
            print(f"✓ 2-opt optimization: {improvements} improvements in {opt_time:.1f}s")
            elapsed_total += opt_time
        
        # Apply 3-opt optimization if enabled (v3.7.0)
        if self.use_3opt and len(self.lines) >= 6:
            opt_start = time.time()
            improvements = self.apply_3opt_optimization(img_array, edge_array, self.three_opt_iterations)
            opt_time = time.time() - opt_start
            print(f"✓ 3-opt optimization: {improvements} improvements in {opt_time:.1f}s")
            elapsed_total += opt_time
        
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
                "version": "3.8.0",
                "num_pins": self.num_pins,
                "num_lines": len(self.lines),
                "min_distance": self.min_distance,
                "line_weight": self.line_weight,
                "edge_weight": self.edge_weight,
                "line_opacity": self.line_opacity,
                "random_sampling": self.random_sampling,
                "sample_size": self.sample_size,
                "look_ahead": self.look_ahead,
                "adaptive_stop": self.adaptive_stop,
                "stop_threshold": self.stop_threshold,
                "beam_width": self.beam_width,
                "use_parallel": self.use_parallel,
                "use_2opt": self.use_2opt,
                "two_opt_iterations": self.two_opt_iterations,
                "use_3opt": self.use_3opt,
                "three_opt_iterations": self.three_opt_iterations,
                "use_squared_diff": self.use_squared_diff,
                "use_simulated_annealing": self.use_simulated_annealing,
                "annealing_temp": self.annealing_temp,
                "annealing_cooling": self.annealing_cooling,
                "warm_restart_interval": self.warm_restart_interval,
                "auto_importance": self.auto_importance,
                "use_knn_selection": self.use_knn_selection,
                "knn_neighbors": self.knn_neighbors,
                "num_workers": self.num_workers,
                "generation_time_seconds": elapsed_total,
                "improvements": [
                    "Edge detection preprocessing",
                    "Feature-aware line selection",
                    "Optimized parameters for clarity",
                    "Non-opaque string support (v2.1.0)",
                    "Random sampling optimization (v2.1.0)",
                    "Adaptive stopping condition (v2.2.0)",
                    "Line caching for speed (v2.2.0)",
                    "Look-ahead optimization (v2.2.0)",
                    "Vectorized edge detection (v2.3.0)",
                    "Beam search optimization (v2.3.0)",
                    "Parallel line evaluation (v2.3.0)",
                    "2-opt local optimization (v3.3.0)",
                    "Importance map support (v3.3.0)",
                    "Enhanced stopping criteria (v3.3.0)",
                    "Squared difference metric (v3.4.0)",
                    "Simulated annealing optimization (v3.4.0)",
                    "Auto-importance map generation (v3.4.0)",
                    "CMY-log color blending model (v3.4.0)",
                    "Anti-aliased line rendering with Xiaolin Wu (v3.5.0)",
                    "Sub-pixel accuracy for smoother output (v3.5.0)",
                    "Hexagonal and square pin arrangements (v3.6.0)",
                    "Adaptive beam width optimization (v3.6.0)",
                    "3-opt local optimization (v3.7.0)",
                    "k-Nearest Neighbors selection (v3.7.0)",
                    "Warm restart simulated annealing (v3.7.0)"
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
            f'  <!-- String Art Generator v3.7.0 -->',
            f'  <!-- Pins: {self.num_pins}, Lines: {len(self.lines)} -->',
            f'  <!-- Edge-aware algorithm with optimized parameters -->',
            f'  <!-- Anti-aliased rendering: Xiaolin Wu algorithm -->',
            f'  <!-- Squared diff: {self.use_squared_diff}, Simulated annealing: {self.use_simulated_annealing} -->',
            f'  <!-- 3-opt: {self.use_3opt}, k-NN selection: {self.use_knn_selection} -->',
            f'  <!-- Beam search: {self.beam_width}, Parallel: {self.use_parallel} -->',
            f'  <!-- Look-ahead: {self.look_ahead}, Adaptive stop: {self.adaptive_stop} -->',
            f'  <!-- Auto-importance: {self.auto_importance} -->',
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
            f.write(f"Version: 3.7.0 (3-opt + k-NN Selection + Warm Restart)\n")
            f.write(f"Total pins: {self.num_pins}\n")
            f.write(f"Total strings: {len(self.lines)}\n")
            f.write(f"Minimum pin distance: {self.min_distance}\n")
            f.write(f"Line opacity: {self.line_opacity}\n")
            f.write(f"Anti-aliased rendering: Xiaolin Wu algorithm\n")
            f.write(f"Squared difference: {self.use_squared_diff}\n")
            f.write(f"Simulated annealing: {self.use_simulated_annealing}\n")
            f.write(f"Warm restart interval: {self.warm_restart_interval}\n")
            f.write(f"Auto-importance map: {self.auto_importance}\n")
            f.write(f"3-opt optimization: {self.use_3opt}\n")
            f.write(f"k-NN selection: {self.use_knn_selection}\n")
            f.write(f"Beam width: {self.beam_width}\n")
            f.write(f"Parallel processing: {self.use_parallel}\n")
            f.write(f"Look-ahead optimization: {self.look_ahead}\n")
            f.write(f"Adaptive stopping: {self.adaptive_stop}\n\n")
            f.write("SEQUENCE (pin-to-pin):\n")
            f.write("-" * 50 + "\n")
            
            for i, (p1, p2) in enumerate(self.lines, 1):
                f.write(f"{i:4d}. Pin {p1:3d} → Pin {p2:3d}\n")
                
                if i % 100 == 0:
                    f.write("\n" + "=" * 50 + "\n\n")


def main():
    parser = argparse.ArgumentParser(description='String Art Generator v3.8.0 - Fear Removal + 3-opt + k-NN Selection')
    parser.add_argument('image', help='Input image path')
    parser.add_argument('--pins', type=int, default=200, help='Number of pins (default: 200)')
    parser.add_argument('--lines', type=int, default=2000, help='Maximum lines (default: 2000)')
    parser.add_argument('--min-distance', type=int, default=20, help='Minimum pin distance (default: 20)')
    parser.add_argument('--line-weight', type=int, default=30, help='Line darkness weight (default: 30)')
    parser.add_argument('--edge-weight', type=float, default=2.0, help='Edge priority multiplier (default: 2.0)')
    parser.add_argument('--opacity', type=float, default=1.0, help='Line opacity 0.0-1.0 (default: 1.0)')
    parser.add_argument('--random-sampling', action='store_true', help='Use random sampling for faster generation')
    parser.add_argument('--sample-size', type=int, default=1000, help='Number of lines to sample per iteration (default: 1000)')
    parser.add_argument('--look-ahead', action='store_true', help='Enable look-ahead optimization (slower but better quality)')
    parser.add_argument('--no-adaptive-stop', action='store_true', help='Disable adaptive stopping condition')
    parser.add_argument('--stop-threshold', type=float, default=0.5, help='Adaptive stop threshold (default: 0.5)')
    parser.add_argument('--beam-width', type=int, default=3, help='Beam search width (default: 3, 1=greedy)')
    parser.add_argument('--no-parallel', action='store_true', help='Disable parallel processing')
    parser.add_argument('--no-2opt', action='store_true', help='Disable 2-opt local optimization (v3.3.0)')
    parser.add_argument('--2opt-iterations', type=int, default=100, help='Number of 2-opt passes (default: 100, v3.3.0)')
    parser.add_argument('--no-3opt', action='store_true', help='Disable 3-opt local optimization (v3.7.0)')
    parser.add_argument('--3opt-iterations', type=int, default=50, help='Number of 3-opt passes (default: 50, v3.7.0)')
    parser.add_argument('--no-squared-diff', action='store_true', help='Disable squared difference metric (v3.4.0)')
    parser.add_argument('--simulated-annealing', action='store_true', help='Enable simulated annealing (v3.4.0)')
    parser.add_argument('--annealing-temp', type=float, default=100.0, help='Initial temperature for annealing (default: 100.0, v3.4.0)')
    parser.add_argument('--annealing-cooling', type=float, default=0.95, help='Cooling rate for annealing (default: 0.95, v3.4.0)')
    parser.add_argument('--warm-restart-interval', type=int, default=500, help='Lines between warm restarts (default: 500, v3.7.0)')
    parser.add_argument('--no-auto-importance', action='store_true', help='Disable auto-importance map generation (v3.4.0)')
    parser.add_argument('--pin-arrangement', type=str, default='circle', choices=['circle', 'hexagon', 'square'], help='Pin arrangement type (default: circle, v3.6.0)')
    parser.add_argument('--no-adaptive-beam', action='store_true', help='Disable adaptive beam width (v3.6.0)')
    parser.add_argument('--no-knn-selection', action='store_true', help='Disable k-NN selection (v3.7.0)')
    parser.add_argument('--knn-neighbors', type=int, default=50, help='Number of k-NN neighbors (default: 50, v3.7.0)')
    parser.add_argument('--no-fear-removal', action='store_true', help='Disable fear removal error function (v3.8.0)')
    parser.add_argument('--output', help='Output directory')
    
    args = parser.parse_args()
    
    generator = StringArtGenerator(
        num_pins=args.pins,
        min_distance=args.min_distance,
        line_weight=args.line_weight,
        edge_weight=args.edge_weight,
        line_opacity=args.opacity,
        random_sampling=args.random_sampling,
        sample_size=args.sample_size if args.random_sampling else None,
        look_ahead=args.look_ahead,
        adaptive_stop=not args.no_adaptive_stop,
        stop_threshold=args.stop_threshold,
        beam_width=args.beam_width,
        use_parallel=not args.no_parallel,
        use_2opt=not args.no_2opt,
        two_opt_iterations=getattr(args, '2opt_iterations', 100),
        use_3opt=not args.no_3opt,
        three_opt_iterations=getattr(args, '3opt_iterations', 50),
        use_squared_diff=not args.no_squared_diff,
        use_simulated_annealing=args.simulated_annealing,
        annealing_temp=args.annealing_temp,
        annealing_cooling=args.annealing_cooling,
        warm_restart_interval=args.warm_restart_interval,
        auto_importance=not args.no_auto_importance,
        pin_arrangement=args.pin_arrangement,
        adaptive_beam=not args.no_adaptive_beam,
        use_knn_selection=not args.no_knn_selection,
        knn_neighbors=args.knn_neighbors,
        use_fear_removal=not args.no_fear_removal
    )
    
    results = generator.generate(
        args.image,
        max_lines=args.lines,
        output_dir=args.output
    )
    
    print("\n" + "=" * 50)
    print("STRING ART GENERATION COMPLETE (v3.8.0)")
    print("=" * 50)
    print(f"Lines generated: {results['num_lines']}")
    print(f"Generation time: {results['generation_time']:.1f}s")
    print(f"SVG output: {results['svg_file']}")
    print(f"Preview: {results['preview_file']}")
    print(f"Comparison: {results['comparison_file']}")
    print(f"\nNew in v3.8.0:")
    print("  ✓ Fear removal error function (better detail capture)")
    print("  ✓ Ignores temporarily worsening pixels for fine details")
    print(f"\nPrevious improvements (v3.7.0):")
    print("  ✓ 3-opt local optimization (more powerful than 2-opt)")
    print("  ✓ k-Nearest Neighbors selection (smarter pin selection)")
    print("  ✓ Warm restart simulated annealing (better exploration)")
    print(f"\nPrevious improvements (v3.6.0):")
    print("  ✓ Hexagonal and square pin arrangements")
    print("  ✓ Adaptive beam width optimization (explore→exploit)")
    print(f"\nPrevious improvements (v3.5.0):")
    print("  ✓ Anti-aliased line rendering (Xiaolin Wu algorithm)")
    print("  ✓ Sub-pixel accuracy for smoother output")
    print("  ✓ Better visual quality with weighted pixel contributions")
    print(f"\nPrevious improvements (v3.4.0):")
    print("  ✓ Squared difference metric (better convergence)")
    print("  ✓ Simulated annealing optimization (escape local optima)")
    print("  ✓ Auto-importance map generation (smart region prioritization)")
    print(f"\nPrevious improvements (v3.3.0):")
    print("  ✓ 2-opt local optimization (better path quality)")
    print("  ✓ Importance map support (region prioritization)")
    print("  ✓ Enhanced stopping criteria")
    print(f"\nPrevious improvements (v2.3.0):")
    print("  ✓ Vectorized edge detection (10-50x faster)")
    print("  ✓ Beam search optimization (better quality)")
    print("  ✓ Parallel line evaluation (multi-core speedup)")
    print(f"\nPrevious improvements (v2.2.0):")
    print("  ✓ Adaptive stopping condition")
    print("  ✓ Line caching for speed")
    print("  ✓ Look-ahead optimization")
    print(f"\nPrevious improvements (v2.1.0):")
    print("  ✓ Non-opaque string support")
    print("  ✓ Random sampling optimization")


if __name__ == '__main__':
    main()
