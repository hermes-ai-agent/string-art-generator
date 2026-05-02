#!/usr/bin/env python3
"""Quick test for v2.3.0 improvements"""

import subprocess
import sys
import time

def run_test(name, args):
    print(f"\n{'='*60}")
    print(f"Test: {name}")
    print(f"{'='*60}")
    start = time.time()
    
    cmd = ["python3", "string_art_generator.py"] + args
    result = subprocess.run(cmd, cwd="/home/amin/string-art", capture_output=True, text=True)
    
    elapsed = time.time() - start
    
    if result.returncode == 0:
        print(f"✓ SUCCESS ({elapsed:.1f}s)")
        # Extract key info from output
        for line in result.stdout.split('\n'):
            if 'Generated' in line or 'Vectorized' in line or 'Beam' in line or 'Parallel' in line:
                print(f"  {line.strip()}")
    else:
        print(f"✗ FAILED ({elapsed:.1f}s)")
        print("STDERR:", result.stderr[-500:] if len(result.stderr) > 500 else result.stderr)
    
    return result.returncode == 0

# Install scipy first
print("Installing scipy...")
subprocess.run([sys.executable, "-m", "pip", "install", "scipy", "-q"], 
               capture_output=True)

# Run tests
tests = [
    ("Beam Search (width=3)", 
     ["test_circle.png", "--pins", "100", "--lines", "200", "--beam-width", "3", 
      "--output", "test_output_v2.3"]),
    
    ("Greedy (beam=1)", 
     ["test_circle.png", "--pins", "100", "--lines", "200", "--beam-width", "1", 
      "--output", "test_output_v2.3_greedy"]),
    
    ("No Parallel", 
     ["test_circle.png", "--pins", "100", "--lines", "200", "--beam-width", "3", 
      "--no-parallel", "--output", "test_output_v2.3_noparallel"]),
]

results = []
for name, args in tests:
    success = run_test(name, args)
    results.append((name, success))

print(f"\n{'='*60}")
print("SUMMARY")
print(f"{'='*60}")
for name, success in results:
    status = "✓ PASS" if success else "✗ FAIL"
    print(f"{status}: {name}")
