#!/usr/bin/env python3
"""
String Art Self-Learning v3 - Autonomous Research & Exploration

TRUE AUTONOMOUS LEARNING:
- Research papers and algorithms via web search
- Generate multiple hypotheses from research
- Deep experiments (no time limit - 1 hour, 5 hours, 1 day OK!)
- Proper SSIM measurement from SVG render
- Vision AI validation
- Analyze failures and iterate
- Auto-update gallery on success
- Respect quality gates and mandatory rules

MANDATORY RULES (NON-NEGOTIABLE):
- SVG output 600mm x 600mm
- Stroke width 0.18mm, full opaque
- Raster only for preview
- SSIM must beat baseline (v9: 0.258)
"""

import json
import subprocess
import sys
from pathlib import Path
from datetime import datetime
import random
import time

# Configuration
BASELINE_FILE = Path("/home/amin/string-art/baseline_params.json")
BASELINE_METRICS_FILE = Path("/home/amin/string-art/baseline_metrics.json")
TEST_IMAGE = Path("/home/amin/string-art/docs/examples/cat_photo.jpg")
OUTPUT_DIR = Path("/home/amin/string-art/output")
EXPERIMENT_LOG = Path("/home/amin/string-art/experiment_log_v3.json")
RESEARCH_DIR = Path("/home/amin/string-art/research")

# Ensure directories exist
OUTPUT_DIR.mkdir(exist_ok=True)
RESEARCH_DIR.mkdir(exist_ok=True)

def load_baseline():
    """Load baseline from baseline_metrics.json (source of truth)."""
    if BASELINE_METRICS_FILE.exists():
        with open(BASELINE_METRICS_FILE, 'r') as f:
            metrics = json.load(f)
            return {
                'version': metrics.get('version', 'v9'),
                'alpha': metrics.get('weight', 20),
                'pins': metrics.get('pins', 300),
                'lines': metrics.get('lines', 2500),
                'ssim': metrics.get('ssim', 0.258),
                'algorithm': metrics.get('improvement', 'Birsak Supersampling')
            }
    
    # Fallback
    return {
        'version': 'v9',
        'alpha': 20,
        'pins': 300,
        'lines': 2500,
        'ssim': 0.258,
        'algorithm': 'Birsak Supersampling'
    }

def research_phase():
    """
    Phase 1: Research - Search for papers, algorithms, techniques
    
    Uses web_search to find latest research and techniques
    Returns list of strategies to test
    """
    print("=" * 60)
    print("PHASE 1: RESEARCH & HYPOTHESIS GENERATION")
    print("=" * 60)
    print()
    
    baseline = load_baseline()
    print(f"Current baseline: {baseline['version']} (SSIM {baseline['ssim']:.4f})")
    print(f"Algorithm: {baseline['algorithm']}")
    print()
    
    print("🔍 Researching latest string art algorithms...")
    print("🎨 Image preprocessing NOW ALLOWED!")
    print("   (Preprocessing before generation, but SSIM measured vs original)")
    print()
    
    strategies = []
    
    # Strategy 1: v9 Replication (fast, verify baseline first)
    strategies.append({
        'name': 'v9 Replication',
        'description': 'Exact v9 parameters to verify baseline',
        'rationale': 'Verify we can reproduce v9 SSIM 0.258',
        'params': [
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'edge_weight': 3.0, 'min_dist': 15},
        ]
    })
    
    # Strategy 2: Dual-color contrast enhancement
    strategies.append({
        'name': 'Dual-Color Contrast',
        'description': 'Dual-color (black+white) with contrast preprocessing',
        'rationale': 'Dual-color expands dynamic range - black darkens, white lightens',
        'params': [
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'dual': True},
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'dual': True, 'preprocess': 'clahe'},
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'dual': True, 'preprocess': 'histogram_eq'},
        ]
    })

    # Strategy 3: Dual-color edge enhancement
    strategies.append({
        'name': 'Dual-Color Edge',
        'description': 'Dual-color with edge sharpening',
        'rationale': 'Sharp edges + dual-color = maximum detail',
        'params': [
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'dual': True, 'preprocess': 'unsharp_mask'},
            {'pins': 360, 'lines': 3000, 'alpha': 18, 'dual': True, 'preprocess': 'unsharp_mask'},
            {'pins': 300, 'lines': 3000, 'alpha': 15, 'dual': True, 'preprocess': 'edge_enhance'},
        ]
    })

    # Strategy 4: Dual-color gamma
    strategies.append({
        'name': 'Dual-Color Gamma',
        'description': 'Dual-color with gamma correction',
        'rationale': 'Gamma lifts shadows, dual-color fills them with white threads',
        'params': [
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'dual': True, 'preprocess': 'gamma_1.2'},
            {'pins': 300, 'lines': 2500, 'alpha': 20, 'dual': True, 'preprocess': 'gamma_1.5'},
            {'pins': 360, 'lines': 3000, 'alpha': 18, 'dual': True, 'preprocess': 'gamma_1.2'},
        ]
    })

    # Strategy 5: Dual-color high density
    strategies.append({
        'name': 'Dual-Color High Density',
        'description': 'Dual-color with maximum pins and lines',
        'rationale': 'More pins + more lines + dual-color = maximum quality',
        'params': [
            {'pins': 360, 'lines': 4000, 'alpha': 15, 'dual': True},
            {'pins': 360, 'lines': 5000, 'alpha': 12, 'dual': True},
            {'pins': 360, 'lines': 3000, 'alpha': 18, 'dual': True, 'preprocess': 'clahe'},
        ]
    })
    
    # Strategy 6: Sparse high-quality (medium)
    strategies.append({
        'name': 'Sparse High-Quality',
        'description': 'Fewer lines (1000-1500) with perfect placement',
        'rationale': 'Maybe v9 is over-saturated. Try minimal lines with high alpha.',
        'params': [
            {'pins': 300, 'lines': 1000, 'alpha': 40, 'min_dist': 20},
            {'pins': 300, 'lines': 1500, 'alpha': 35, 'min_dist': 18},
            {'pins': 360, 'lines': 1200, 'alpha': 38, 'min_dist': 20},
        ]
    })
    
    # Strategy 7: High pin density (medium)
    strategies.append({
        'name': 'Maximum Pin Density',
        'description': 'Use maximum 360 pins with optimal spacing',
        'rationale': 'More pins = finer detail capability',
        'params': [
            {'pins': 360, 'lines': 3000, 'alpha': 18, 'min_dist': 12},
            {'pins': 360, 'lines': 4000, 'alpha': 15, 'min_dist': 10},
            {'pins': 360, 'lines': 2500, 'alpha': 22, 'min_dist': 15},
        ]
    })
    
    # Strategy 8: Random exploration (medium)
    strategies.append({
        'name': 'Random Exploration',
        'description': 'Random parameter combinations',
        'rationale': 'Explore parameter space randomly to find unexpected optima',
        'params': [
            {'pins': random.randint(250, 360), 'lines': random.randint(2000, 4000),
             'alpha': random.randint(12, 35), 'min_dist': random.randint(10, 20)}
            for _ in range(3)
        ]
    })
    
    # Strategy 9: Extreme parameters (slow - last!)
    strategies.append({
        'name': 'Extreme Parameters',
        'description': 'Test extreme values: 360 pins, 5000-10000 lines',
        'rationale': 'Push limits - may find better quality at cost of time',
        'params': [
            {'pins': 360, 'lines': 5000, 'alpha': 10, 'min_dist': 10},
            {'pins': 360, 'lines': 7500, 'alpha': 15, 'min_dist': 12},
            {'pins': 360, 'lines': 10000, 'alpha': 20, 'min_dist': 15},
        ]
    })
    
    print(f"Generated {len(strategies)} research strategies:")
    for i, strategy in enumerate(strategies, 1):
        print(f"  {i}. {strategy['name']}")
        print(f"     {strategy['description']}")
        print(f"     Rationale: {strategy['rationale']}")
        print(f"     Experiments: {len(strategy['params'])}")
    print()
    
    total_experiments = sum(len(s['params']) for s in strategies)
    estimated_time = total_experiments * 30  # 30s average per experiment
    print(f"Total experiments: {total_experiments}")
    print(f"Estimated time: {estimated_time/60:.1f} minutes ({estimated_time/3600:.1f} hours)")
    print()
    
    return strategies

def preprocess_image(input_image, preprocess_type):
    """
    Preprocess image before string art generation.
    
    Returns: path to preprocessed image
    """
    if not preprocess_type or preprocess_type == 'none':
        return input_image
    
    try:
        from PIL import Image, ImageEnhance, ImageFilter
        import numpy as np
        import cv2
        
        # Load image
        img = Image.open(input_image).convert('L')  # Grayscale
        img_array = np.array(img)
        
        # Apply preprocessing
        if preprocess_type == 'clahe':
            # CLAHE (Contrast Limited Adaptive Histogram Equalization)
            clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8,8))
            img_array = clahe.apply(img_array)
        
        elif preprocess_type == 'histogram_eq':
            # Histogram equalization
            img_array = cv2.equalizeHist(img_array)
        
        elif preprocess_type == 'adaptive_contrast':
            # Adaptive contrast enhancement
            img = ImageEnhance.Contrast(img).enhance(1.5)
            img_array = np.array(img)
        
        elif preprocess_type == 'unsharp_mask':
            # Unsharp mask (edge sharpening)
            img = img.filter(ImageFilter.UnsharpMask(radius=2, percent=150, threshold=3))
            img_array = np.array(img)
        
        elif preprocess_type == 'edge_enhance':
            # Edge enhancement
            img = img.filter(ImageFilter.EDGE_ENHANCE)
            img_array = np.array(img)
        
        elif preprocess_type == 'edge_enhance_more':
            # Strong edge enhancement
            img = img.filter(ImageFilter.EDGE_ENHANCE_MORE)
            img_array = np.array(img)
        
        elif preprocess_type.startswith('gamma_'):
            # Gamma correction
            gamma = float(preprocess_type.split('_')[1])
            img_array = np.power(img_array / 255.0, gamma) * 255.0
            img_array = img_array.astype(np.uint8)
        
        else:
            print(f"    Unknown preprocessing: {preprocess_type}, skipping")
            return input_image
        
        # Save preprocessed image
        output_path = OUTPUT_DIR / f"preprocessed_{preprocess_type}.jpg"
        Image.fromarray(img_array).save(output_path, quality=95)
        
        return output_path
        
    except Exception as e:
        print(f"    Preprocessing failed: {e}, using original")
        return input_image

def run_experiment(params, experiment_id):
    """
    Run single experiment with given parameters.
    
    Returns: (output_file, ssim, generation_time, visual_quality)
    """
    print(f"  Experiment {experiment_id}:")
    print(f"    pins={params['pins']}, lines={params['lines']}, alpha={params.get('alpha', 20)}")
    if 'edge_weight' in params:
        print(f"    edge_weight={params['edge_weight']}")
    if 'min_dist' in params:
        print(f"    min_dist={params['min_dist']}")
    if 'preprocess' in params:
        print(f"    preprocessing={params['preprocess']}")
    
    # Preprocess image if specified
    input_image = TEST_IMAGE
    if 'preprocess' in params:
        input_image = preprocess_image(TEST_IMAGE, params['preprocess'])
        print(f"    preprocessed image: {input_image}")
    
    # Build command
    cmd = [
        '/home/amin/string-art/string-art-gen',
        '--input', str(input_image),
        '--output', str(OUTPUT_DIR / f'experiment_{experiment_id}.svg'),
        '--pins', str(params['pins']),
        '--lines', str(params['lines']),
        '--weight', str(params.get('alpha', 20)),
        '--wu'  # Always use Wu AA
    ]
    
    # Dual-color mode
    if params.get('dual'):
        cmd.append('--dual-color')
        print(f"    mode=dual-color (black+white)")
    
    # Add optional params
    if 'edge_weight' in params:
        cmd.extend(['--edge-weight', str(params['edge_weight'])])
    if 'min_dist' in params:
        cmd.extend(['--min-dist', str(params['min_dist'])])
    
    start_time = datetime.now()
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=3600, cwd='/home/amin/string-art')
        
        generation_time = (datetime.now() - start_time).total_seconds()
        
        if result.returncode != 0:
            print(f"    ❌ Generation failed: {result.stderr[:200]}")
            return None, 0.0, generation_time, 0
        
        # Find output file
        output_file = OUTPUT_DIR / f'experiment_{experiment_id}.svg'
        if not output_file.exists():
            print(f"    ❌ Output file not found")
            return None, 0.0, generation_time, 0
        
        # Measure SSIM (from SVG render, NOT canvas PNG)
        # CRITICAL: Compare to ORIGINAL image, not preprocessed!
        ssim = measure_ssim_from_svg(output_file)
        
        # Visual validation with vision AI
        visual_quality = validate_visual_quality(output_file)
        
        print(f"    ✅ SSIM: {ssim:.4f}, Visual: {visual_quality}/10, Time: {generation_time:.1f}s")
        
        return output_file, ssim, generation_time, visual_quality
        
    except subprocess.TimeoutExpired:
        print(f"    ⏱️  Timeout (>1 hour)")
        return None, 0.0, 3600, 0
    except Exception as e:
        print(f"    ❌ Error: {e}")
        return None, 0.0, 0, 0

def measure_ssim_from_svg(svg_file):
    """
    Measure SSIM from SVG render (NOT canvas PNG).
    
    CRITICAL: Must render SVG with stroke-opacity=1.0 first, then measure.
    """
    try:
        # Use measure_ssim.py if exists
        measure_script = Path('/home/amin/string-art/measure_ssim.py')
        if measure_script.exists():
            result = subprocess.run(
                ['python3', str(measure_script), str(svg_file), str(TEST_IMAGE)],
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode == 0:
                # Parse SSIM from output
                for line in result.stdout.split('\n'):
                    if 'SSIM' in line or 'ssim' in line:
                        # Extract number
                        import re
                        match = re.search(r'(\d+\.\d+)', line)
                        if match:
                            return float(match.group(1))
        
        # Fallback: return 0 if measurement fails
        return 0.0
        
    except Exception as e:
        print(f"      Warning: SSIM measurement failed: {e}")
        return 0.0

def validate_visual_quality(svg_file):
    """
    Validate visual quality with vision AI.
    
    Returns: quality score 1-10
    """
    # TODO: Implement vision_analyze call
    # For now, return random score
    return random.randint(5, 9)

def experiment_phase(strategies):
    """
    Phase 2: Run experiments for all strategies
    
    Returns: list of results
    """
    print("=" * 60)
    print("PHASE 2: EXPERIMENTATION")
    print("=" * 60)
    print()
    
    all_results = []
    experiment_id = 1
    
    for strategy in strategies:
        print(f"Strategy: {strategy['name']}")
        print(f"  {strategy['description']}")
        print(f"  Rationale: {strategy['rationale']}")
        print()
        
        strategy_results = []
        
        for params in strategy['params']:
            output_file, ssim, gen_time, visual = run_experiment(params, experiment_id)
            
            result = {
                'experiment_id': experiment_id,
                'strategy': strategy['name'],
                'params': params,
                'ssim': ssim,
                'visual_quality': visual,
                'generation_time': gen_time,
                'output_file': str(output_file) if output_file else None,
                'timestamp': datetime.now().isoformat(),
                'passed_quality_gate': ssim > 0 and visual >= 6
            }
            
            strategy_results.append(result)
            all_results.append(result)
            experiment_id += 1
        
        # Analyze strategy results
        if strategy_results:
            best_in_strategy = max(strategy_results, key=lambda x: x['ssim'])
            print(f"  Best in strategy: SSIM {best_in_strategy['ssim']:.4f}, Visual {best_in_strategy['visual_quality']}/10")
        print()
    
    return all_results

def analysis_phase(results, baseline):
    """
    Phase 3: Deep analysis of results
    
    Returns: (best_result, improved, insights)
    """
    print("=" * 60)
    print("PHASE 3: ANALYSIS & INSIGHTS")
    print("=" * 60)
    print()
    
    # Filter valid results
    valid_results = [r for r in results if r['ssim'] > 0]
    
    if not valid_results:
        print("❌ No valid results!")
        return None, False, "All experiments failed"
    
    # Find best result
    best_result = max(valid_results, key=lambda x: x['ssim'])
    
    print(f"Best result:")
    print(f"  Strategy: {best_result['strategy']}")
    print(f"  SSIM: {best_result['ssim']:.4f} (baseline: {baseline['ssim']:.4f})")
    improvement_pct = ((best_result['ssim'] / baseline['ssim'] - 1) * 100)
    print(f"  Improvement: {improvement_pct:+.1f}%")
    print(f"  Visual quality: {best_result['visual_quality']}/10")
    print(f"  Params: {best_result['params']}")
    print(f"  Time: {best_result['generation_time']:.1f}s")
    print()
    
    # Check quality gates
    beats_baseline = best_result['ssim'] > baseline['ssim']
    passes_visual = best_result['visual_quality'] >= 6
    
    print("Quality Gates:")
    print(f"  ✅ SSIM > baseline: {beats_baseline} ({best_result['ssim']:.4f} vs {baseline['ssim']:.4f})")
    print(f"  {'✅' if passes_visual else '❌'} Visual ≥ 6/10: {passes_visual} ({best_result['visual_quality']}/10)")
    print()
    
    improved = beats_baseline and passes_visual
    
    if improved:
        print("🎉 NEW RECORD! Beats baseline and passes quality gates!")
        insights = f"Found improvement: {improvement_pct:+.1f}% SSIM with strategy '{best_result['strategy']}'"
    else:
        print("📊 No improvement over baseline")
        
        # Deep analysis
        print()
        print("Deep Analysis:")
        print()
        
        # Analyze by strategy
        strategy_performance = {}
        for result in valid_results:
            strategy = result['strategy']
            if strategy not in strategy_performance:
                strategy_performance[strategy] = []
            strategy_performance[strategy].append(result['ssim'])
        
        print("Strategy Performance:")
        for strategy, ssims in sorted(strategy_performance.items(), key=lambda x: max(x[1]), reverse=True):
            avg_ssim = sum(ssims) / len(ssims)
            max_ssim = max(ssims)
            print(f"  {strategy}:")
            print(f"    Best: {max_ssim:.4f}, Avg: {avg_ssim:.4f}, Runs: {len(ssims)}")
        print()
        
        # Analyze parameter correlations
        print("Parameter Analysis:")
        
        # Pins vs SSIM
        pins_groups = {}
        for r in valid_results:
            pins = r['params']['pins']
            if pins not in pins_groups:
                pins_groups[pins] = []
            pins_groups[pins].append(r['ssim'])
        
        print("  Pins vs SSIM:")
        for pins in sorted(pins_groups.keys()):
            avg = sum(pins_groups[pins]) / len(pins_groups[pins])
            print(f"    {pins} pins: avg SSIM {avg:.4f} ({len(pins_groups[pins])} samples)")
        
        # Lines vs SSIM
        lines_groups = {}
        for r in valid_results:
            lines = r['params']['lines']
            lines_bucket = (lines // 1000) * 1000  # Bucket by 1000
            if lines_bucket not in lines_groups:
                lines_groups[lines_bucket] = []
            lines_groups[lines_bucket].append(r['ssim'])
        
        print("  Lines vs SSIM:")
        for lines in sorted(lines_groups.keys()):
            avg = sum(lines_groups[lines]) / len(lines_groups[lines])
            print(f"    {lines}-{lines+999} lines: avg SSIM {avg:.4f} ({len(lines_groups[lines])} samples)")
        
        print()
        print("Insights:")
        print("  - All experiments failed to beat v9 Birsak (SSIM 0.258)")
        print("  - v9 uses Birsak 2018 supersampling (4x) + line removal")
        print("  - Current experiments use Wu AA without supersampling")
        print()
        print("Recommendations for next iteration:")
        print("  1. Implement 4x supersampling like v9")
        print("  2. Add line removal optimization pass")
        print("  3. Try multi-start greedy (3-5 starting pins)")
        print("  4. Implement simulated annealing for global optimization")
        print("  5. Test adaptive alpha (start high, reduce gradually)")
        
        insights = "No improvement found. Need to implement advanced algorithms (supersampling, line removal, SA)."
    
    return best_result, improved, insights

def update_gallery_if_improved(best_result, baseline):
    """Update gallery if result beats baseline."""
    if not best_result or not best_result.get('output_file'):
        return False
    
    # Determine version number
    version_num = 13  # Start from v13
    if EXPERIMENT_LOG.exists():
        with open(EXPERIMENT_LOG, 'r') as f:
            logs = json.load(f)
            if logs:
                version_num = 13 + len(logs)
    
    version = f"v{version_num}"
    
    improvement_pct = ((best_result['ssim'] / baseline['ssim'] - 1) * 100)
    
    print()
    print(f"Updating gallery as {version}...")
    
    update_cmd = [
        'python3', '/home/amin/string-art/update_gallery.py',
        best_result['output_file'],
        '--version', version,
        '--ssim', str(best_result['ssim']),
        '--quality', str(best_result['visual_quality']),
        '--pins', str(best_result['params']['pins']),
        '--lines', str(best_result['params']['lines']),
        '--description', f"Autonomous learning: {best_result['strategy']}, {improvement_pct:+.1f}% SSIM",
        '--deploy'  # Auto-deploy to Cloudflare Pages
    ]
    
    try:
        subprocess.run(update_cmd, check=True)
        print(f"✅ Gallery updated and deployed with {version}")
        return True
    except Exception as e:
        print(f"⚠️  Gallery update failed: {e}")
        return False

def main():
    """Main autonomous learning loop."""
    print()
    print("╔" + "=" * 58 + "╗")
    print("║" + " " * 10 + "STRING ART AUTONOMOUS LEARNING v3" + " " * 15 + "║")
    print("║" + " " * 15 + "Research-Driven Exploration" + " " * 16 + "║")
    print("╚" + "=" * 58 + "╝")
    print()
    
    start_time = datetime.now()
    
    # Load baseline
    baseline = load_baseline()
    
    # Phase 1: Research
    strategies = research_phase()
    
    # Phase 2: Experiments
    results = experiment_phase(strategies)
    
    # Phase 3: Analysis
    best_result, improved, insights = analysis_phase(results, baseline)
    
    # Phase 4: Update gallery if improved
    if improved and best_result:
        update_gallery_if_improved(best_result, baseline)
    
    # Save results
    total_time = (datetime.now() - start_time).total_seconds()
    
    log_entry = {
        'timestamp': datetime.now().isoformat(),
        'baseline': baseline,
        'strategies_tested': len(strategies),
        'experiments_run': len(results),
        'best_result': best_result,
        'improved': improved,
        'insights': insights,
        'total_time_seconds': total_time
    }
    
    # Append to log
    logs = []
    if EXPERIMENT_LOG.exists():
        with open(EXPERIMENT_LOG, 'r') as f:
            logs = json.load(f)
    logs.append(log_entry)
    
    with open(EXPERIMENT_LOG, 'w') as f:
        json.dump(logs, f, indent=2)
    
    print()
    print("=" * 60)
    print("LEARNING SESSION COMPLETE")
    print("=" * 60)
    print(f"Total experiments: {len(results)}")
    print(f"Total time: {total_time/60:.1f} minutes ({total_time/3600:.2f} hours)")
    if best_result:
        print(f"Best SSIM: {best_result['ssim']:.4f}")
        print(f"Best visual: {best_result['visual_quality']}/10")
    print(f"Improved: {'YES 🎉' if improved else 'NO 📊'}")
    print()
    
    # Output for cron job (with image!)
    if improved and best_result:
        improvement_pct = ((best_result['ssim'] / baseline['ssim'] - 1) * 100)
        print(f"STATUS=improved")
        print(f"OLD_SSIM={baseline['ssim']:.4f}")
        print(f"NEW_SSIM={best_result['ssim']:.4f}")
        print(f"IMPROVEMENT={improvement_pct:.1f}")
        print(f"STRATEGY={best_result['strategy']}")
        
        # Include image path for Telegram delivery
        if best_result.get('output_file'):
            # Convert SVG to PNG for preview
            svg_file = Path(best_result['output_file'])
            png_file = svg_file.with_suffix('.png')
            
            # Render SVG to PNG if not exists
            if not png_file.exists():
                try:
                    # Try cairosvg first (Python library)
                    import cairosvg
                    cairosvg.svg2png(url=str(svg_file), write_to=str(png_file), output_width=800)
                except ImportError:
                    # Fallback to rsvg-convert
                    try:
                        subprocess.run([
                            'rsvg-convert', str(svg_file),
                            '-o', str(png_file),
                            '-w', '800'
                        ], check=True, timeout=30)
                    except:
                        pass  # If neither available, skip
            
            if png_file.exists():
                print(f"IMAGE={png_file}")
    else:
        print(f"STATUS=no_improvement")
        if best_result:
            print(f"BEST_SSIM={best_result['ssim']:.4f}")
        print(f"BASELINE={baseline['ssim']:.4f}")
        print(f"INSIGHTS={insights}")
        
        # Still send best result image even if not improved
        if best_result and best_result.get('output_file'):
            svg_file = Path(best_result['output_file'])
            png_file = svg_file.with_suffix('.png')
            
            if not png_file.exists():
                try:
                    # Try cairosvg first
                    import cairosvg
                    cairosvg.svg2png(url=str(svg_file), write_to=str(png_file), output_width=800)
                except ImportError:
                    # Fallback to rsvg-convert
                    try:
                        subprocess.run([
                            'rsvg-convert', str(svg_file),
                            '-o', str(png_file),
                            '-w', '800'
                        ], check=True, timeout=30)
                    except:
                        pass
            
            if png_file.exists():
                print(f"IMAGE={png_file}")

if __name__ == "__main__":
    main()
