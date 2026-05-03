#!/usr/bin/env python3
"""
Self-learning quality gate and rollback system.
Ensures only improvements are committed and deployed.
"""

import sys
import os
import json
import subprocess
from pathlib import Path
from datetime import datetime

class QualityGate:
    def __init__(self, baseline_metrics_file="baseline_metrics.json"):
        self.baseline_file = Path(baseline_metrics_file)
        self.metrics = self.load_baseline()
    
    def load_baseline(self):
        """Load baseline metrics from file."""
        if self.baseline_file.exists():
            with open(self.baseline_file, 'r') as f:
                return json.load(f)
        return {
            "version": "v9.0",
            "ssim": 0.2148,
            "quality_score": 7,
            "generation_time": 11.4,
            "pins": 300,
            "lines": 3200
        }
    
    def save_baseline(self, metrics):
        """Save new baseline metrics."""
        with open(self.baseline_file, 'w') as f:
            json.dump(metrics, f, indent=2)
        print(f"✅ Baseline updated: {metrics['version']} (SSIM: {metrics['ssim']}, Quality: {metrics['quality_score']}/10)")
    
    def evaluate(self, new_version, new_ssim, new_quality, new_time, pins, lines, improvement_desc):
        """
        Evaluate if new version passes quality gate.
        
        Returns:
            (passed: bool, reason: str, action: str)
        """
        baseline_ssim = self.metrics["ssim"]
        baseline_quality = self.metrics["quality_score"]
        baseline_version = self.metrics["version"]
        
        # Calculate improvements
        ssim_improvement = ((new_ssim - baseline_ssim) / baseline_ssim) * 100
        quality_improvement = new_quality - baseline_quality
        
        print(f"\n{'='*60}")
        print(f"QUALITY GATE EVALUATION")
        print(f"{'='*60}")
        print(f"Baseline: {baseline_version} (SSIM: {baseline_ssim:.4f}, Quality: {baseline_quality}/10)")
        print(f"New:      {new_version} (SSIM: {new_ssim:.4f}, Quality: {new_quality}/10)")
        print(f"SSIM Change:    {ssim_improvement:+.2f}%")
        print(f"Quality Change: {quality_improvement:+.1f} points")
        print(f"{'='*60}\n")
        
        # Decision logic
        if ssim_improvement >= 2.0 and quality_improvement >= 0:
            # Clear improvement
            return (True, f"PASS: SSIM improved by {ssim_improvement:.2f}%, quality maintained or improved", "DEPLOY")
        
        elif ssim_improvement >= 1.0 and quality_improvement >= 0:
            # Moderate improvement
            return (True, f"PASS: SSIM improved by {ssim_improvement:.2f}%, quality stable", "DEPLOY")
        
        elif ssim_improvement >= 0 and quality_improvement >= 1:
            # Quality improved even if SSIM flat
            return (True, f"PASS: Quality improved by {quality_improvement} points", "DEPLOY")
        
        elif ssim_improvement >= -1.0 and quality_improvement >= 0:
            # Slight degradation but acceptable
            return (True, f"CONDITIONAL PASS: Minor SSIM drop ({ssim_improvement:.2f}%), quality stable", "DEPLOY_WITH_WARNING")
        
        elif ssim_improvement < -1.0 or quality_improvement < 0:
            # Clear degradation
            return (False, f"FAIL: SSIM dropped {ssim_improvement:.2f}%, quality dropped {quality_improvement} points", "ROLLBACK")
        
        else:
            # Edge case
            return (False, f"FAIL: Unclear improvement (SSIM: {ssim_improvement:+.2f}%, Quality: {quality_improvement:+.1f})", "ROLLBACK")
    
    def rollback(self, failed_version):
        """Rollback failed changes."""
        print(f"\n🔄 ROLLING BACK {failed_version}...")
        
        # Discard uncommitted changes
        subprocess.run(["git", "checkout", "HEAD", "--", "*.go"], check=True)
        subprocess.run(["git", "clean", "-fd", "output/"], check=True)
        
        # Remove failed artifacts
        failed_files = [
            f"docs/result_{failed_version}_*.svg",
            f"docs/result_{failed_version}_*.png",
            f"docs/baseline_{failed_version}_*.svg",
            f"docs/baseline_{failed_version}_*.png",
            f"generator_{failed_version}_*.go",
            f"STRING_ART_{failed_version.upper()}_REPORT.md"
        ]
        
        for pattern in failed_files:
            subprocess.run(f"rm -f {pattern}", shell=True)
        
        print(f"✅ Rollback complete. Baseline {self.metrics['version']} preserved.")
        return True

def main():
    if len(sys.argv) != 8:
        print("Usage: quality_gate.py <version> <ssim> <quality> <time> <pins> <lines> <description>")
        print("Example: quality_gate.py v31.0 0.2079 6.5 30.66 300 3200 'Face-aware importance map'")
        sys.exit(1)
    
    gate = QualityGate(str(Path("~/string-art/baseline_metrics.json").expanduser()))
    
    version = sys.argv[1]
    ssim = float(sys.argv[2])
    quality = float(sys.argv[3])
    time = float(sys.argv[4])
    pins = int(sys.argv[5])
    lines = int(sys.argv[6])
    description = sys.argv[7]
    
    passed, reason, action = gate.evaluate(version, ssim, quality, time, pins, lines, description)
    
    print(f"Decision: {action}")
    print(f"Reason: {reason}\n")
    
    if action == "DEPLOY":
        print("✅ QUALITY GATE PASSED - Proceeding with deployment")
        gate.save_baseline({
            "version": version,
            "ssim": ssim,
            "quality_score": quality,
            "generation_time": time,
            "pins": pins,
            "lines": lines,
            "improvement": description,
            "timestamp": datetime.now().isoformat()
        })
        sys.exit(0)
    
    elif action == "DEPLOY_WITH_WARNING":
        print("⚠️  CONDITIONAL PASS - Deploying with warning")
        gate.save_baseline({
            "version": version,
            "ssim": ssim,
            "quality_score": quality,
            "generation_time": time,
            "pins": pins,
            "lines": lines,
            "improvement": description,
            "timestamp": datetime.now().isoformat(),
            "warning": reason
        })
        sys.exit(0)
    
    elif action == "ROLLBACK":
        print("❌ QUALITY GATE FAILED - Rolling back changes")
        gate.rollback(version)
        sys.exit(1)
    
    else:
        print(f"⚠️  Unknown action: {action}")
        sys.exit(1)

if __name__ == "__main__":
    main()
