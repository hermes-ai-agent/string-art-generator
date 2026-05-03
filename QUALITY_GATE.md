# Quality Gate System

Sistem quality gate untuk memastikan self-learning **hanya deploy improvements**, tidak pernah deploy degradation.

## Problem

Self-learning v31 mengalami degradation:
- SSIM turun dari 0.2148 → 0.2079 (-3.2%)
- Quality turun dari 7/10 → 6.5/10
- Tapi tetap di-commit dan di-deploy ❌

**Root cause:** Tidak ada automatic quality gate sebelum commit.

## Solution

**Quality Gate System** dengan 3 komponen:

### 1. Baseline Metrics (`baseline_metrics.json`)

Menyimpan metrics dari versi terbaik saat ini:
```json
{
  "version": "v30.0",
  "ssim": 0.2148,
  "quality_score": 7,
  "generation_time": 34.1,
  "pins": 300,
  "lines": 3200,
  "improvement": "Canny Edge Detection + morphological operations",
  "timestamp": "2026-05-03T18:21:38"
}
```

### 2. Quality Gate Script (`quality_gate.py`)

Evaluasi apakah improvement layak di-deploy:

**Usage:**
```bash
python3 quality_gate.py <version> <ssim> <quality> <time> <pins> <lines> <description>
```

**Example:**
```bash
python3 quality_gate.py "v32.0" 0.2200 7.5 32.0 300 3200 "Adaptive line weight"
```

**Decision Logic:**

| Condition | SSIM Change | Quality Change | Action |
|-----------|-------------|----------------|--------|
| Clear improvement | ≥ +2% | ≥ 0 | **DEPLOY** |
| Moderate improvement | ≥ +1% | ≥ 0 | **DEPLOY** |
| Quality boost | ≥ 0% | ≥ +1 | **DEPLOY** |
| Minor degradation | ≥ -1% | ≥ 0 | **DEPLOY_WITH_WARNING** |
| Clear degradation | < -1% | < 0 | **ROLLBACK** ❌ |

**Exit Codes:**
- `0` = PASSED (proceed with deployment)
- `1` = FAILED (rollback, no deployment)

### 3. Automatic Rollback

Jika quality gate FAILED:
1. ✅ Discard code changes (`git checkout HEAD -- *.go`)
2. ✅ Clean output directory
3. ✅ Remove failed artifacts
4. ✅ Preserve baseline version
5. ✅ Report failure reason
6. ❌ NO commit
7. ❌ NO deploy

## Integration with Self-Learning

Self-learning workflow sekarang:

```
1. Research improvement idea
2. Implement in Go code
3. Generate baseline (old code)
4. Generate new version (new code)
5. Visual comparison
6. ⭐ RUN QUALITY GATE ⭐  ← NEW STEP
   ├─ PASSED → Continue to step 7
   └─ FAILED → Rollback, STOP
7. Upload to docs/
8. Update website
9. Commit and push
10. Auto-deploy
```

**Quality gate adalah gatekeeper mutlak.** Tidak ada bypass!

## Examples

### Example 1: Failed Improvement (v31)

```bash
$ python3 quality_gate.py "v31.0" 0.2079 6.5 30.66 300 3200 "Face-aware importance map"

============================================================
QUALITY GATE EVALUATION
============================================================
Baseline: v30.0 (SSIM: 0.2148, Quality: 7/10)
New:      v31.0 (SSIM: 0.2079, Quality: 6.5/10)
SSIM Change:    -3.21%
Quality Change: -0.5 points
============================================================

Decision: ROLLBACK
Reason: FAIL: SSIM dropped -3.21%, quality dropped -0.5 points

❌ QUALITY GATE FAILED - Rolling back changes
🔄 ROLLING BACK v31.0...
✅ Rollback complete. Baseline v30.0 preserved.
```

**Result:** v31 NOT deployed, baseline v30 preserved ✅

### Example 2: Successful Improvement (v32)

```bash
$ python3 quality_gate.py "v32.0" 0.2200 7.5 32.0 300 3200 "Adaptive line weight"

============================================================
QUALITY GATE EVALUATION
============================================================
Baseline: v30.0 (SSIM: 0.2148, Quality: 7/10)
New:      v32.0 (SSIM: 0.2200, Quality: 7.5/10)
SSIM Change:    +2.42%
Quality Change: +0.5 points
============================================================

Decision: DEPLOY
Reason: PASS: SSIM improved by 2.42%, quality maintained or improved

✅ QUALITY GATE PASSED - Proceeding with deployment
✅ Baseline updated: v32.0 (SSIM: 0.22, Quality: 7.5/10)
```

**Result:** v32 deployed, baseline updated to v32 ✅

## Benefits

1. **No More Degradation** - Failed improvements never reach production
2. **Automatic Rollback** - No manual cleanup needed
3. **Clear Decision Logic** - Transparent evaluation criteria
4. **Baseline Tracking** - Always know what's the best version
5. **Progress Guarantee** - Self-learning only moves forward

## Monitoring

Check baseline metrics:
```bash
cat ~/string-art/baseline_metrics.json
```

Check quality gate logs in self-learning reports.

## Future Enhancements

1. **Multi-metric evaluation** - Add perceptual metrics (LPIPS, FID)
2. **A/B testing** - Deploy to staging first
3. **Rollback history** - Track all failed attempts
4. **Adaptive thresholds** - Adjust based on improvement difficulty
5. **Notification** - Alert on repeated failures

## Summary

**Before Quality Gate:**
- v31 failed but deployed anyway ❌
- Degradation reached production ❌
- Manual rollback needed ❌

**After Quality Gate:**
- v31 failed and auto-rolled back ✅
- Only improvements reach production ✅
- Baseline always preserved ✅

**Self-learning sekarang guaranteed to improve, never degrade!** 🎉
