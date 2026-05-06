#!/usr/bin/env python3
"""
Self-Learning V5: Creative Research-Driven Evolution
- Web research untuk discover teknik baru
- Knowledge base persistent (belajar dari semua run sebelumnya)
- Creative combination generator (tidak pernah repeat combo yang sama)
- 4 strategi: Evolution (30%), Research (30%), Creative (30%), Extreme (10%)
"""

import json
import subprocess
import os
import sys
import random
import hashlib
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Tuple, Optional

# Paths
BASE_DIR = Path("/home/amin/string-art")
KNOWLEDGE_BASE = BASE_DIR / "knowledge_base_v5.json"
RESEARCH_CACHE = BASE_DIR / "research_cache_v5.json"
OUTPUT_DIR = BASE_DIR / "output"
GENERATOR = BASE_DIR / "string-art-gen"
INPUT_IMAGE = BASE_DIR / "docs/examples/cat_photo.jpg"

# Constants
BASELINE_SSIM = 0.22  # Target to beat (QUALITY GATE - DO NOT CHANGE)
BEST_EVER_SSIM = 0.2316  # V13 Kitchen Sink
MANDATORY_STROKE = 0.18  # mm, non-negotiable

class KnowledgeBase:
    """Persistent knowledge across ALL self-learning runs"""
    
    def __init__(self):
        self.data = self._load()
    
    def _load(self) -> Dict:
        if KNOWLEDGE_BASE.exists():
            with open(KNOWLEDGE_BASE) as f:
                data = json.load(f)
                # No migration needed - keep SSIM as-is
                return data
        
        return {
            "successful_techniques": [],  # SSIM >= baseline
            "failed_techniques": [],      # SSIM < baseline
            "research_insights": [],      # From web research
            "creative_combos": [],        # Novel combinations tried
            "best_result": {
                "ssim": BEST_EVER_SSIM,
                "version": "v13_kitchen_sink",
                "params": {
                    "pins": 360,
                    "lines": 5000,
                    "alpha": 18,
                    "flags": ["birsak", "dual-color", "sa", "look-ahead", "random-sampling"]
                }
            },
            "run_count": 0,
            "last_updated": None
        }
    
    def save(self):
        self.data["last_updated"] = datetime.now().isoformat()
        with open(KNOWLEDGE_BASE, 'w') as f:
            json.dump(self.data, f, indent=2)
    
    def add_result(self, technique: Dict, ssim: float, svg_path: str):
        """Add result and learn from it"""
        entry = {
            "technique": technique,
            "ssim": ssim,
            "svg_path": str(svg_path),
            "timestamp": datetime.now().isoformat(),
            "run_number": self.data["run_count"] + 1
        }
        
        if ssim >= BASELINE_SSIM:
            self.data["successful_techniques"].append(entry)
            # Update best if better (higher SSIM)
            if ssim > self.data["best_result"]["ssim"]:
                self.data["best_result"] = {
                    "ssim": ssim,
                    "version": technique.get("name", "unknown"),
                    "params": technique
                }
                print(f"🏆 NEW BEST RESULT: SSIM {ssim:.4f}")
        else:
            self.data["failed_techniques"].append(entry)
        
        self.data["run_count"] += 1
        self.save()
    
    def get_successful_techniques(self) -> List[Dict]:
        """Get all successful techniques, sorted by SSIM (higher is better)"""
        return sorted(
            self.data["successful_techniques"],
            key=lambda x: x["ssim"],
            reverse=True  # Higher SSIM is better
        )
    
    def get_top_n(self, n: int = 5) -> List[Dict]:
        """Get top N successful techniques"""
        return self.get_successful_techniques()[:n]
    
    def has_tried(self, technique_hash: str) -> bool:
        """Check if we've tried this exact combination before"""
        all_techniques = (
            self.data["successful_techniques"] + 
            self.data["failed_techniques"] +
            self.data["creative_combos"]
        )
        return any(t.get("hash") == technique_hash for t in all_techniques)
    
    def add_research_insight(self, insight: Dict):
        """Add insight from web research"""
        self.data["research_insights"].append({
            **insight,
            "timestamp": datetime.now().isoformat()
        })
        self.save()
    
    def add_creative_combo(self, combo: Dict):
        """Track creative combinations tried"""
        self.data["creative_combos"].append({
            **combo,
            "timestamp": datetime.now().isoformat()
        })
        self.save()


class ResearchEngine:
    """Web research to discover new techniques"""
    
    def __init__(self):
        self.cache = self._load_cache()
    
    def _load_cache(self) -> Dict:
        if RESEARCH_CACHE.exists():
            with open(RESEARCH_CACHE) as f:
                return json.load(f)
        return {"searches": {}, "last_updated": None}
    
    def _save_cache(self):
        self.cache["last_updated"] = datetime.now().isoformat()
        with open(RESEARCH_CACHE, 'w') as f:
            json.dump(self.cache, f, indent=2)
    
    def search(self, query: str) -> List[str]:
        """Search web for techniques (cached)"""
        # Check cache first
        if query in self.cache["searches"]:
            cached = self.cache["searches"][query]
            # Cache valid for 7 days
            cache_age = (datetime.now() - datetime.fromisoformat(cached["timestamp"])).days
            if cache_age < 7:
                print(f"📚 Using cached research: {query}")
                return cached["insights"]
        
        print(f"🔍 Researching: {query}")
        
        # Research topics
        topics = [
            "string art optimization algorithms",
            "line art generation techniques",
            "simulated annealing for art",
            "genetic algorithms for image approximation",
            "multi-scale image processing",
            "edge detection for line art",
            "adaptive sampling strategies",
            "dual-color string art",
            "thread tension optimization"
        ]
        
        insights = []
        
        # Simulate research (in production, this would call web_search tool)
        # For now, generate insights based on known techniques
        research_insights = {
            "optimization": [
                "Multi-start simulated annealing with temperature scheduling",
                "Genetic algorithms with crossover and mutation",
                "Particle swarm optimization for pin placement",
                "Gradient descent on line selection"
            ],
            "preprocessing": [
                "CLAHE (Contrast Limited Adaptive Histogram Equalization)",
                "Bilateral filtering for edge preservation",
                "Unsharp masking for detail enhancement",
                "Multi-scale decomposition (Laplacian pyramid)"
            ],
            "sampling": [
                "Importance sampling based on edge strength",
                "Stratified sampling for uniform coverage",
                "Blue noise sampling for even distribution",
                "Adaptive sampling density based on image complexity"
            ],
            "multi-threading": [
                "Parallel line evaluation with thread pool",
                "Lock-free data structures for concurrent access",
                "SIMD vectorization for pixel operations",
                "GPU acceleration with CUDA/OpenCL"
            ],
            "advanced": [
                "Look-ahead search (evaluate N steps ahead)",
                "Beam search with multiple candidates",
                "Monte Carlo tree search for line selection",
                "Reinforcement learning for policy optimization"
            ]
        }
        
        # Extract relevant insights
        for category, techniques in research_insights.items():
            if any(keyword in query.lower() for keyword in category.split()):
                insights.extend(techniques)
        
        # Cache results
        self.cache["searches"][query] = {
            "insights": insights,
            "timestamp": datetime.now().isoformat()
        }
        self._save_cache()
        
        return insights
    
    def discover_new_techniques(self) -> List[Dict]:
        """Research and generate new technique ideas"""
        queries = [
            "string art optimization",
            "line art generation algorithms",
            "image approximation techniques"
        ]
        
        all_insights = []
        for query in queries:
            insights = self.search(query)
            all_insights.extend(insights)
        
        # Convert insights to technique specs
        techniques = []
        for insight in all_insights[:3]:  # Top 3 insights
            technique = self._insight_to_technique(insight)
            if technique:
                techniques.append(technique)
        
        return techniques
    
    def _insight_to_technique(self, insight: str) -> Optional[Dict]:
        """Convert research insight to executable technique"""
        technique = {
            "name": f"research_{hashlib.md5(insight.encode()).hexdigest()[:8]}",
            "source": "research",
            "insight": insight,
            "params": {
                "pins": random.choice([280, 300, 320, 340, 360]),  # Max 360
                "lines": random.randint(3000, 6000),
                "alpha": random.randint(10, 40),  # weight 10-40 (darkness)
                "flags": []
            }
        }
        
        # Map insights to flags
        insight_lower = insight.lower()
        
        if "simulated annealing" in insight_lower or "temperature" in insight_lower:
            technique["params"]["flags"].append("sa")
        
        if "look-ahead" in insight_lower or "search" in insight_lower:
            technique["params"]["flags"].append("look-ahead")
        
        if "sampling" in insight_lower:
            technique["params"]["flags"].append("random-sampling")
        
        if "edge" in insight_lower or "contrast" in insight_lower:
            technique["params"]["flags"].append("high-contrast")
            technique["params"]["edge_weight"] = random.uniform(1.5, 3.0)
        
        if "multi" in insight_lower or "dual" in insight_lower:
            technique["params"]["flags"].append("dual-color")
        
        # Always try Birsak supersampling (but only 20% chance to avoid timeout)
        if random.random() > 0.8:
            technique["params"]["flags"].append("birsak")
        
        return technique


class CreativeCombinator:
    """Generate novel technique combinations"""
    
    def __init__(self, kb: KnowledgeBase):
        self.kb = kb
    
    def generate_novel_combo(self) -> Dict:
        """Generate a combination we've never tried before"""
        max_attempts = 100
        
        for _ in range(max_attempts):
            combo = self._random_combo()
            combo_hash = self._hash_technique(combo)
            
            if not self.kb.has_tried(combo_hash):
                combo["hash"] = combo_hash
                return combo
        
        # If all combos tried, generate extreme random
        return self._extreme_random_combo()
    
    def _random_combo(self) -> Dict:
        """Generate random combination of techniques"""
        # Available flags (removed birsak - too slow, reduced SA probability)
        all_flags = [
            "dual-color",
            "look-ahead",
            "random-sampling",
            "high-contrast",
            "enhanced"
        ]
        
        # Add SA only 30% of the time (it's slow)
        if random.random() < 0.3:
            all_flags.append("sa")
        
        # Randomly select 2-5 flags
        num_flags = random.randint(2, 5)
        flags = random.sample(all_flags, num_flags)
        
        combo = {
            "name": f"creative_{'_'.join(sorted(flags))}",
            "source": "creative",
            "params": {
                "pins": random.choice([280, 300, 320, 340, 360]),  # Max 360
                "lines": random.randint(2500, 7000),
                "alpha": random.randint(10, 50),  # weight 10-50
                "flags": flags
            }
        }
        
        # Add optional params
        if "high-contrast" in flags:
            combo["params"]["edge_weight"] = random.uniform(1.2, 3.5)
        
        if random.random() > 0.5:
            combo["params"]["min_dist"] = random.randint(3, 12)
        
        return combo
    
    def _extreme_random_combo(self) -> Dict:
        """Extreme experimental combination"""
        return {
            "name": f"extreme_{random.randint(1000, 9999)}",
            "source": "extreme",
            "params": {
                "pins": random.choice([200, 250, 300, 350, 360]),  # Max 360
                "lines": random.randint(1000, 10000),
                "alpha": random.randint(5, 80),  # weight 5-80 (extreme range)
                "flags": random.sample([
                    "dual-color", "sa", "look-ahead",
                    "random-sampling", "high-contrast", "enhanced"
                ], k=random.randint(1, 6)),  # Removed birsak (too slow)
                "edge_weight": random.uniform(0.5, 5.0),
                "min_dist": random.randint(1, 20)
            }
        }
    
    def evolve_from_best(self, technique: Dict) -> Dict:
        """Evolve from a successful technique"""
        evolved = {
            "name": f"evolved_{technique['technique'].get('name', 'unknown')}",
            "source": "evolution",
            "parent": technique['technique'].get('name'),
            "parent_ssim": technique['ssim'],
            "params": technique['technique']['params'].copy()
        }
        
        # Mutate parameters
        mutations = random.randint(1, 3)
        
        for _ in range(mutations):
            mutation_type = random.choice([
                "pins", "lines", "alpha", "add_flag", "remove_flag", "tweak_param"
            ])
            
            if mutation_type == "pins":
                evolved["params"]["pins"] = max(200, min(360,
                    evolved["params"]["pins"] + random.randint(-40, 40)
                ))
            
            elif mutation_type == "lines":
                evolved["params"]["lines"] = max(1000, min(10000,
                    evolved["params"]["lines"] + random.randint(-1000, 1000)
                ))
            
            elif mutation_type == "alpha":
                evolved["params"]["alpha"] = max(5, min(80,
                    evolved["params"]["alpha"] + random.randint(-10, 10)
                ))
            
            elif mutation_type == "add_flag":
                available = ["birsak", "dual-color", "sa", "look-ahead",
                           "random-sampling", "high-contrast", "enhanced"]
                current = evolved["params"].get("flags", [])
                new_flags = [f for f in available if f not in current]
                if new_flags:
                    evolved["params"]["flags"] = current + [random.choice(new_flags)]
            
            elif mutation_type == "remove_flag":
                flags = evolved["params"].get("flags", [])
                if len(flags) > 1:
                    flags.remove(random.choice(flags))
                    evolved["params"]["flags"] = flags
            
            elif mutation_type == "tweak_param":
                if "edge_weight" in evolved["params"]:
                    evolved["params"]["edge_weight"] += random.uniform(-0.5, 0.5)
                elif random.random() > 0.5:
                    evolved["params"]["edge_weight"] = random.uniform(1.5, 3.0)
        
        evolved["hash"] = self._hash_technique(evolved)
        return evolved
    
    def _hash_technique(self, technique: Dict) -> str:
        """Generate unique hash for technique"""
        params = technique.get("params", {})
        key = f"{params.get('pins')}_{params.get('lines')}_{params.get('alpha'):.2f}_{'_'.join(sorted(params.get('flags', [])))}"
        return hashlib.md5(key.encode()).hexdigest()


class StringArtGenerator:
    """Execute string art generation"""
    
    def __init__(self):
        self.generator = GENERATOR
        self.input_image = INPUT_IMAGE
        self.output_dir = OUTPUT_DIR
    
    def generate(self, technique: Dict) -> Tuple[Optional[str], Optional[float]]:
        """Generate string art and return (svg_path, ssim)"""
        params = technique.get("params", {})
        
        # Build command
        output_name = f"{technique.get('name', 'test')}.svg"
        output_path = self.output_dir / output_name
        
        cmd = [
            str(self.generator),
            "--input", str(self.input_image),
            "--output", str(output_path),
            "--pins", str(params.get("pins", 360)),
            "--lines", str(params.get("lines", 3000)),
            "--weight", str(int(params.get("alpha", 15))),  # alpha -> weight (darkness)
            "--opacity", str(1.0)  # Always opaque (mandatory)
        ]
        
        # Add flags
        for flag in params.get("flags", []):
            cmd.append(f"--{flag}")
        
        # Add optional params
        if "edge_weight" in params:
            cmd.extend(["--edge-weight", str(params["edge_weight"])])
        
        if "min_dist" in params:
            cmd.extend(["--min-dist", str(params["min_dist"])])
        
        print(f"🎨 Generating:, flush=True) {technique.get('name')}")
        print(f"   Command: {' '.join(cmd)}")
        
        try:
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=1800  # 30 min timeout (Birsak + SA bisa lama)
            )
            
            if result.returncode != 0:
                error_msg = result.stderr if result.stderr else result.stdout
                print(f"❌ Generation failed: {error_msg}")
                return None, None
            
            # Calculate SSIM from SVG output
            ssim = self._calculate_ssim(output_path)
            
            if ssim is None:
                print(f"⚠️  Could not calculate SSIM")
                return str(output_path), None
            
            print(f"✅ Generated:, flush=True) SSIM {ssim:.4f}")
            return str(output_path), ssim
            
        except subprocess.TimeoutExpired:
            print(f"⏱️  Generation timeout")
            return None, None
        except Exception as e:
            print(f"❌ Error: {e}")
            return None, None
    
    def _calculate_ssim(self, svg_path: Path) -> Optional[float]:
        """Calculate SSIM using canvas PNG generated by generator"""
        try:
            # Generator outputs canvas PNG automatically
            # Format: output/xxx.svg → output/xxx_canvas.png
            canvas_path = svg_path.parent / f"{svg_path.stem}_canvas.png"
            
            if not canvas_path.exists():
                print(f"⚠️  Canvas PNG not found: {canvas_path}")
                return None
            
            # Calculate SSIM using Python
            from skimage.metrics import structural_similarity as ssim
            from PIL import Image
            import numpy as np
            
            # Load images (resize to same size for comparison)
            img1 = Image.open(self.input_image).convert('L').resize((800, 800))
            img2 = Image.open(canvas_path).convert('L').resize((800, 800))
            
            # Convert to numpy arrays
            arr1 = np.array(img1)
            arr2 = np.array(img2)
            
            # Calculate SSIM
            ssim_value = ssim(arr1, arr2, data_range=255)
            
            return ssim_value
            
        except Exception as e:
            print(f"⚠️  SSIM calculation error: {e}")
            return None


class SelfLearningV5:
    """Main self-learning orchestrator"""
    
    def __init__(self):
        self.kb = KnowledgeBase()
        self.research = ResearchEngine()
        self.creative = CreativeCombinator(self.kb)
        self.generator = StringArtGenerator()
    
    def run(self):
        """Execute one self-learning iteration"""
        print("=" * 60, flush=True)
        print(f"🧠 Self-Learning V5 - Run #{self.kb.data['run_count'] + 1}", flush=True)
        print(f"📊 Best SSIM so far: {self.kb.data['best_result']['ssim']:.4f}", flush=True)
        print(f"✅ Successful techniques: {len(self.kb.data['successful_techniques'])}", flush=True)
        print(f"❌ Failed techniques: {len(self.kb.data['failed_techniques'])}", flush=True)
        print("=" * 60, flush=True)
        
        # Choose strategy
        strategy = self._choose_strategy()
        print(f"🎯 Strategy:, flush=True) {strategy}")
        
        # Generate technique based on strategy
        technique = self._generate_technique(strategy)
        
        if technique is None:
            print("❌ Could not generate technique")
            return
        
        print(f"🔬 Technique:, flush=True) {technique.get('name')}")
        print(f"   Source: {technique.get('source')}")
        print(f"   Params: {json.dumps(technique.get('params'), indent=2)}")
        
        # Generate and evaluate
        svg_path, ssim = self.generator.generate(technique)
        
        if svg_path and ssim:
            # Learn from result
            self.kb.add_result(technique, ssim, svg_path)
            
            # Output for cron
            print(f"\n{'='*60}")
            print(f"📈 RESULT: SSIM {ssim:.4f}")
            print(f"📁 IMAGE={svg_path}")
            print(f"{'='*60}")
            
            # Check if we beat baseline
            if ssim >= BASELINE_SSIM:
                print(f"🎉 SUCCESS! Beat baseline {BASELINE_SSIM}")
                if ssim > self.kb.data['best_result']['ssim']:
                    print(f"🏆 NEW RECORD! Previous: {self.kb.data['best_result']['ssim']:.4f}")
            else:
                print(f"📉 Below baseline. Keep learning...")
        else:
            print("❌ Generation failed")
    
    def _choose_strategy(self) -> str:
        """Choose learning strategy based on distribution"""
        rand = random.random()
        
        if rand < 0.30:
            return "evolution"  # 30% - evolve from best
        elif rand < 0.60:
            return "research"   # 30% - research-driven
        elif rand < 0.90:
            return "creative"   # 30% - novel combinations
        else:
            return "extreme"    # 10% - extreme experiments
    
    def _generate_technique(self, strategy: str) -> Optional[Dict]:
        """Generate technique based on strategy"""
        
        if strategy == "evolution":
            # Evolve from top performers
            top = self.kb.get_top_n(5)
            if not top:
                print("⚠️  No successful techniques yet, falling back to creative")
                return self.creative.generate_novel_combo()
            
            parent = random.choice(top)
            return self.creative.evolve_from_best(parent)
        
        elif strategy == "research":
            # Research-driven new techniques
            techniques = self.research.discover_new_techniques()
            if not techniques:
                print("⚠️  No research insights, falling back to creative")
                return self.creative.generate_novel_combo()
            
            technique = random.choice(techniques)
            self.kb.add_research_insight({
                "technique": technique,
                "insight": technique.get("insight")
            })
            return technique
        
        elif strategy == "creative":
            # Novel combinations
            combo = self.creative.generate_novel_combo()
            self.kb.add_creative_combo(combo)
            return combo
        
        elif strategy == "extreme":
            # Extreme experiments
            return self.creative._extreme_random_combo()
        
        return None


def main():
    """Entry point"""
    learner = SelfLearningV5()
    learner.run()


if __name__ == "__main__":
    main()
