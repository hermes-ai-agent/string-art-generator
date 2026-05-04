#!/usr/bin/env python3
"""
String Art Self-Learning v2 - Parameter Optimization
Automatically experiments with different parameters to improve SSIM quality.

This script:
1. Loads current baseline parameters
2. Generates variations (alpha, pins, lines)
3. Tests each variation
4. Saves improved parameters if SSIM increases
"""

import json
import subprocess
import sys
from pathlib import Path
from datetime import datetime
import random

# Configuration
BASELINE_FILE = Path("/home/amin/string-art/baseline_params.json")
BASELINE_METRICS_FILE = Path("/home/amin/string-art/baseline_metrics.json")
TEST_IMAGE = Path("/home/amin/string-art/docs/examples/cat_photo.jpg")
OUTPUT_DIR = Path("/home/amin/string-art/output")
EXPERIMENT_LOG = Path("/home/amin/string-art/experiment_log.json")

# Default baseline from baseline_metrics.json (v10)
DEFAULT_BASELINE = {
    "alpha": 20,      # weight parameter for Go binary
    "pins": 300,
    "lines": 2500,
    "ssim": 0.1959,   # v10 baseline SSIM
    "last_updated": None
}

def load_baseline():
    """Load current baseline parameters from baseline_params.json or baseline_metrics.json."""
    # Try baseline_params.json first
    if BASELINE_FILE.exists():
        with open(BASELINE_FILE, 'r') as f:
            params = json.load(f)
            # Validate SSIM is not zero
            if params.get('ssim', 0) > 0:
                return params
            else:
                print(f"⚠️  baseline_params.json has invalid SSIM ({params.get('ssim')}), loading from baseline_metrics.json")
    
    # Fallback to baseline_metrics.json
    if BASELINE_METRICS_FILE.exists():
        with open(BASELINE_METRICS_FILE, 'r') as f:
            metrics = json.load(f)
            params = {
                'alpha': metrics.get('weight', 20),
                'pins': metrics.get('pins', 300),
                'lines': metrics.get('lines', 2500),
                'ssim': metrics.get('ssim', 0.1959),
                'last_updated': metrics.get('timestamp')
            }
            # Save to baseline_params.json for next run
            save_baseline(params)
            print(f"✅ Loaded baseline from baseline_metrics.json: SSIM={params['ssim']:.4f}")
            return params
    
    # Last resort: use default
    print(f"⚠️  No baseline files found, using default: SSIM={DEFAULT_BASELINE['ssim']:.4f}")
    return DEFAULT_BASELINE.copy()

def save_baseline(params):
    """Save new baseline parameters."""
    params['last_updated'] = datetime.now().isoformat()
    with open(BASELINE_FILE, 'w') as f:
        json.dump(params, f, indent=2)

def log_experiment(params, ssim, improved):
    """Log experiment results."""
    log_entry = {
        "timestamp": datetime.now().isoformat(),
        "params": params,
        "ssim": ssim,
        "improved": improved
    }
    
    logs = []
    if EXPERIMENT_LOG.exists():
        with open(EXPERIMENT_LOG, 'r') as f:
            logs = json.load(f)
    
    logs.append(log_entry)
    
    # Keep only last 100 experiments
    logs = logs[-100:]
    
    with open(EXPERIMENT_LOG, 'w') as f:
        json.dump(logs, f, indent=2)

def generate_variation(baseline):
    """Generate parameter variation for testing."""
    # Strategy: small random variations around current best
    variations = {
        'alpha': [
            baseline['alpha'],
            max(10, baseline['alpha'] - 5),
            min(50, baseline['alpha'] + 5),
            max(10, int(baseline['alpha'] * 0.9)),
            min(50, int(baseline['alpha'] * 1.1))
        ],
        'pins': [
            baseline['pins'],
            max(100, baseline['pins'] - 20),
            min(300, baseline['pins'] + 20),
            max(100, int(baseline['pins'] * 0.9)),
            min(300, int(baseline['pins'] * 1.1))
        ],
        'lines': [
            baseline['lines'],
            max(1000, baseline['lines'] - 500),
            min(5000, baseline['lines'] + 500),
            max(1000, int(baseline['lines'] * 0.9)),
            min(5000, int(baseline['lines'] * 1.1))
        ]
    }
    
    return {
        'alpha': random.choice(variations['alpha']),
        'pins': random.choice(variations['pins']),
        'lines': random.choice(variations['lines'])
    }

def run_generation(params):
    """Run string art generation with given parameters using Go binary."""
    # Clear old output files
    import glob
    for f in glob.glob(str(OUTPUT_DIR / "cat_photo_*.svg")):
        Path(f).unlink(missing_ok=True)
    
    # Use Go binary with Wu anti-aliased mode (V11)
    cmd = [
        '/home/amin/string-art/string-art-gen',
        '--input', str(TEST_IMAGE),
        '--output', str(OUTPUT_DIR / 'cat_photo_stringart.svg'),
        '--pins', str(params['pins']),
        '--lines', str(params['lines']),
        '--weight', str(params['alpha']),
        '--wu'  # Use V11 Wu anti-aliased generator
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=600, cwd='/home/amin/string-art')
        if result.returncode != 0:
            print(f"Generation failed: {result.stderr}")
            return None
        
        # Find the generated SVG file
        svg_files = list(OUTPUT_DIR.glob("cat_photo_*.svg"))
        if not svg_files:
            print("No SVG output found")
            return None
        
        # Return the most recent file
        return max(svg_files, key=lambda p: p.stat().st_mtime)
    except subprocess.TimeoutExpired:
        print("Generation timeout")
        return None
    except Exception as e:
        print(f"Generation error: {e}")
        return None

def measure_ssim(output_file):
    """Measure SSIM of generated output."""
    cmd = [
        'python3', '/home/amin/string-art/measure_ssim.py',
        str(output_file),
        str(TEST_IMAGE)
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        if result.returncode != 0:
            print(f"SSIM measurement failed: {result.stderr}")
            return None
        
        # Parse SSIM from output
        for line in result.stdout.split('\n'):
            if line.startswith('MEASURED_SSIM='):
                return float(line.split('=')[1])
        
        return None
    except Exception as e:
        print(f"SSIM measurement error: {e}")
        return None

def main():
    """Main self-learning loop."""
    print("=== String Art Self-Learning v2 ===")
    print(f"Test image: {TEST_IMAGE}")
    print()
    
    # Load baseline
    baseline = load_baseline()
    print(f"Current baseline:")
    print(f"  alpha={baseline['alpha']}, pins={baseline['pins']}, lines={baseline['lines']}")
    print(f"  SSIM: {baseline['ssim']:.4f}")
    print()
    
    # Generate variation
    params = generate_variation(baseline)
    print(f"Testing variation:")
    print(f"  alpha={params['alpha']}, pins={params['pins']}, lines={params['lines']}")
    print()
    
    # Run generation
    print("Generating string art...")
    output_file = run_generation(params)
    if output_file is None:
        print("❌ Generation failed")
        log_experiment(params, 0.0, False)
        sys.exit(1)
    
    # Measure SSIM
    print("Measuring SSIM...")
    ssim = measure_ssim(output_file)
    if ssim is None:
        print("❌ SSIM measurement failed")
        log_experiment(params, 0.0, False)
        sys.exit(1)
    
    print(f"Result: SSIM = {ssim:.4f}")
    print()
    
    # Determine version number (auto-increment from experiment log)
    version_num = 12  # Start from v12
    if EXPERIMENT_LOG.exists():
        with open(EXPERIMENT_LOG, 'r') as f:
            logs = json.load(f)
            if logs:
                version_num = 12 + len(logs)
    
    version = f"v{version_num}"
    
    # Check if improved
    if ssim > baseline['ssim']:
        improvement = ((ssim - baseline['ssim']) / baseline['ssim'] * 100) if baseline['ssim'] > 0 else 100
        print(f"🎉 IMPROVEMENT FOUND!")
        print(f"   Old SSIM: {baseline['ssim']:.4f}")
        print(f"   New SSIM: {ssim:.4f}")
        print(f"   Gain: +{improvement:.1f}%")
        print()
        print(f"New parameters:")
        print(f"   alpha={params['alpha']}, pins={params['pins']}, lines={params['lines']}")
        
        # Save new baseline
        new_baseline = {
            'alpha': params['alpha'],
            'pins': params['pins'],
            'lines': params['lines'],
            'ssim': ssim
        }
        save_baseline(new_baseline)
        log_experiment(params, ssim, True)
        
        # Update gallery with new version
        print()
        print(f"Updating gallery as {version}...")
        update_cmd = [
            'python3', '/home/amin/string-art/update_gallery.py',
            str(output_file),
            '--version', version,
            '--ssim', str(ssim),
            '--quality', str(min(10, int(ssim * 40))),  # Quality score based on SSIM
            '--pins', str(params['pins']),
            '--lines', str(params['lines']),
            '--description', f"Self-learning: alpha={params['alpha']}, +{improvement:.1f}% SSIM",
            '--deploy'  # Auto-deploy to Cloudflare Pages
        ]
        try:
            subprocess.run(update_cmd, check=True)
            print(f"✅ Gallery updated with {version}")
        except Exception as e:
            print(f"⚠️  Gallery update failed: {e}")
        
        # Output for cron job
        print()
        print(f"STATUS=improved")
        print(f"VERSION={version}")
        print(f"OLD_SSIM={baseline['ssim']:.4f}")
        print(f"NEW_SSIM={ssim:.4f}")
        print(f"IMPROVEMENT={improvement:.1f}")
        print(f"ALPHA={params['alpha']}")
        print(f"PINS={params['pins']}")
        print(f"LINES={params['lines']}")
    else:
        print(f"📊 No improvement")
        print(f"   Baseline: {baseline['ssim']:.4f}")
        print(f"   Current:  {ssim:.4f}")
        log_experiment(params, ssim, False)
        
        # Still update gallery but mark as failed
        print()
        print(f"Updating gallery as {version} (failed)...")
        update_cmd = [
            'python3', '/home/amin/string-art/update_gallery.py',
            str(output_file),
            '--version', version,
            '--ssim', str(ssim),
            '--quality', str(min(10, int(ssim * 40))),
            '--pins', str(params['pins']),
            '--lines', str(params['lines']),
            '--description', f"Self-learning: alpha={params['alpha']}, no improvement",
            '--failed',
            '--deploy'  # Auto-deploy even for failed experiments
        ]
        try:
            subprocess.run(update_cmd, check=True)
            print(f"✅ Gallery updated with {version} (marked as failed)")
        except Exception as e:
            print(f"⚠️  Gallery update failed: {e}")
        
        # Output for cron job
        print()
        print(f"STATUS=no_improvement")
        print(f"VERSION={version}")
        print(f"SSIM={ssim:.4f}")
        print(f"BASELINE={baseline['ssim']:.4f}")
        print(f"ALPHA={params['alpha']}")
        print(f"PINS={params['pins']}")
        print(f"LINES={params['lines']}")

if __name__ == "__main__":
    main()
