#!/usr/bin/env python3
"""
Cron wrapper for Self-Learning V5
Checks system idle, runs learning, outputs result
"""

import subprocess
import sys
import os

def check_system_idle():
    """Check if system is idle (CPU < 50%)"""
    try:
        result = subprocess.run(
            ["top", "-bn1"],
            capture_output=True,
            text=True,
            timeout=5
        )
        
        for line in result.stdout.split('\n'):
            if '%Cpu' in line or 'CPU' in line:
                # Extract idle percentage
                import re
                match = re.search(r'(\d+\.\d+)\s*id', line)
                if match:
                    idle = float(match.group(1))
                    return idle > 50  # System idle if >50% idle
        
        return True  # Default to idle if can't determine
    except:
        return True  # Default to idle on error

def main():
    # Check if system is idle
    if not check_system_idle():
        print("STATUS=skip")
        print("REASON=System busy (CPU > 50%)")
        sys.exit(0)
    
    # Run self-learning V5
    script_path = os.path.join(os.path.dirname(__file__), "self_learning_v5_creative.py")
    
    try:
        result = subprocess.run(
            ["python3", script_path],
            capture_output=True,
            text=True,
            timeout=2400  # 40 min timeout (30 min generation + 10 min overhead)
        )
        
        # Forward output
        print(result.stdout)
        
        if result.stderr:
            print(result.stderr, file=sys.stderr)
        
        sys.exit(result.returncode)
        
    except subprocess.TimeoutExpired:
        print("STATUS=timeout")
        print("REASON=Learning took > 15 minutes")
        sys.exit(1)
    except Exception as e:
        print(f"STATUS=error")
        print(f"REASON={e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
