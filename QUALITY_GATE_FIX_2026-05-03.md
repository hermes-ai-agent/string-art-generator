# Quality Gate System - Comprehensive Fix (2026-05-03 22:00)

## Problem Diagnosis

### Root Cause: Self-Learning Claiming Fake Metrics

**Evidence:**
- `baseline_metrics.json` claimed v34 SSIM: 0.4917 (+90% improvement!)
- `results_manifest.json` showed v34 actual SSIM: 0.2 (-23% degradation)
- Vision analysis rated latest output: 6.5/10 (not 7.5/10 as claimed)
- User complaint: "hasil terus degrading" - confirmed by data

**How it happened:**
1. Self-learning generated code improvements
2. Claimed SSIM improvements without measuring
3. Quality gate accepted claimed metrics
4. Degraded versions deployed to production
5. Baseline corrupted (v30 SSIM 0.2148 instead of v9 SSIM 0.258)

### Secondary Issues

1. **LLM Provider Error** - Cron job failing with "No LLM provider configured"
2. **Model Switching Degradation** - User noted: "Opus mahal tapi bagus, Sonnet jadi jelek"
3. **No Measurement Enforcement** - Quality gate accepted claims, didn't require proof

## Solutions Implemented

### 1. Baseline Reset ✅

**File:** `~/string-art/baseline_metrics.json`

```json
{
  "version": "v9.0",
  "ssim": 0.258,
  "quality_score": 6,
  "generation_time": 11.4,
  "pins": 300,
  "lines": 2500,
  "improvement": "Birsak Supersampling + Enhanced Scoring",
  "timestamp": "2026-05-03T10:14:00",
  "note": "BASELINE RESET 2026-05-03 22:00 - User confirmed v9 is GOOD"
}
```

**Rationale:** v9 is user-confirmed GOOD version. v30-v34 are degraded.

### 2. Quality Gate Hardcoded Baseline Fix ✅

**File:** `~/string-art/quality_gate.py`

Changed fallback baseline from v30 metrics (SSIM 0.2148) to v9 metrics (SSIM 0.258).

### 3. SSIM Measurement Script ✅

**File:** `~/string-art/measure_ssim.py`

**Purpose:** MANDATORY measurement of actual SSIM. No claims allowed.

**Usage:**
```bash
python3 measure_ssim.py output/result.svg cat_photo.jpg
# Output: MEASURED_SSIM=0.258000
```

**Integration:** Self-learning MUST call this script and use measured value for quality gate.

### 4. Self-Learning Prompt Update ✅

**Cron Job:** `2b3bea51d716` (Self-Learning Idle Only)

**Key Changes:**
- MANDATORY workflow with quality gate checkpoint
- NO BYPASS allowed
- measure_ssim.py required (no claims)
- Visual validation required
- Quality gate EXIT 0 = deploy, EXIT 1 = rollback
- Modify existing file only (no new generator_vN files)

**Workflow:**
```
Research → Implement → Build → Baseline → New → 
MEASURE SSIM → Visual → Quality Gate → 
IF PASSED: Upload → Website → Gallery → Commit → Deploy
IF FAILED: Rollback → Report → STOP
```

### 5. Dependencies Check

**Required Python packages:**
```bash
pip3 install pillow numpy scikit-image
```

**Required system tools:**
```bash
# ImageMagick for SVG→PNG conversion
sudo apt install imagemagick
```

## Verification Steps

### 1. Test SSIM Measurement

```bash
cd ~/string-art
python3 measure_ssim.py docs/OUTPUT.svg cat_photo.jpg
```

Expected output: `MEASURED_SSIM=0.XXXXXX`

### 2. Test Quality Gate with Measured SSIM

```bash
cd ~/string-art
python3 quality_gate.py "v35.0" 0.260 7 10.5 300 2500 "Test improvement"
```

Expected: PASS (SSIM improved from 0.258 to 0.260)

### 3. Test Quality Gate with Degradation

```bash
cd ~/string-art
python3 quality_gate.py "v35.0" 0.200 8 10.5 300 2500 "Fake improvement"
```

Expected: FAIL (SSIM dropped from 0.258 to 0.200, -22.5%)

### 4. Verify Cron Job Configuration

```bash
hermes cron list
# Check job 2b3bea51d716:
# - workdir: /home/amin/string-art
# - script: learning_simple.py
# - enabled_toolsets: web, file, terminal, memory
```

### 5. Manual Trigger Test

```bash
hermes cron run 2b3bea51d716
```

Watch for:
- ✅ Script executes (learning_simple.py)
- ✅ LLM provider works
- ✅ Quality gate enforced
- ✅ Measurement required

## Expected Behavior Going Forward

### Self-Learning Session (Success Path)

1. **Research** - Pick improvement from priority list
2. **Implement** - Modify `generator_v9_birsak.go`
3. **Build** - `go build` (must succeed)
4. **Baseline** - Generate with OLD code, save metrics
5. **New** - Generate with NEW code
6. **Measure** - `measure_ssim.py` (actual SSIM)
7. **Visual** - `vision_analyze` (quality 1-10)
8. **Quality Gate** - `quality_gate.py` with measured values
   - IF SSIM improved ≥1% → DEPLOY
   - IF SSIM stable + quality improved → DEPLOY
   - IF SSIM degraded >1% → ROLLBACK
9. **Upload** - Copy to docs/ with timestamp
10. **Website** - `update_website.py`
11. **Gallery** - `generate_manifest.py`
12. **Deploy** - Git commit + push
13. **Report** - Telegram notification with results

### Self-Learning Session (Failure Path)

1-8. Same as success path
9. **Quality Gate FAILED** → EXIT 1
10. **Rollback** - `git checkout HEAD -- *.go`
11. **Cleanup** - Remove failed artifacts
12. **Report** - Telegram notification: "v35 FAILED: SSIM dropped -5.2%"
13. **STOP** - No upload, no commit, no deploy

### Degradation Prevention

**Quality Gate Decision Logic:**

| SSIM Change | Quality Change | Decision |
|-------------|----------------|----------|
| +2% or more | Any | ✅ DEPLOY |
| +1% to +2% | Any | ✅ DEPLOY |
| 0% to +1% | ≥0 | ✅ DEPLOY |
| 0% to +1% | <0 | ⚠️ DEPLOY_WITH_WARNING |
| -1% to 0% | +1 or more | ⚠️ DEPLOY_WITH_WARNING |
| -1% to 0% | <+1 | ❌ ROLLBACK |
| -1% to -5% | Any | ❌ ROLLBACK |
| <-5% | Any | ❌ ROLLBACK (critical) |

**Key Principle:** SSIM is PRIMARY (objective), quality score is SECONDARY (subjective).

## Monitoring

### Check Baseline Status

```bash
cat ~/string-art/baseline_metrics.json
```

Should show: v9.0, SSIM 0.258

### Check Recent Learning Sessions

```bash
ls -lht ~/.hermes/cron/output/2b3bea51d716/ | head -5
```

### Check Gallery

https://string-art-generator-2ih.pages.dev/gallery.html

Should show v9 as baseline, v30-v34 marked as degraded.

### Check Quality Gate Logs

```bash
cd ~/string-art
git log --grep="FAILED" --oneline
```

Failed attempts should be visible in git history.

## Remaining Issues

### 1. LLM Provider Error in Cron

**Symptom:** "No LLM provider configured" in cron execution

**Possible Causes:**
- Workdir not set correctly
- Environment variables not loaded
- Model config not accessible from cron context

**Next Steps:**
- Test manual cron run: `hermes cron run 2b3bea51d716`
- Check cron output logs for detailed error
- Verify model config in cron context

### 2. Model Quality Variance

**User Observation:** "Opus mahal tapi bagus, Sonnet jadi jelek"

**Implication:** Different models produce different quality results, even with same code.

**Solution:** Quality gate is model-agnostic - it measures ACTUAL output quality, not code quality. If Sonnet produces worse SSIM, quality gate will reject it.

### 3. Historical Degraded Versions

**Status:** v30-v34 are in git history and gallery with degraded metrics.

**Options:**
- Keep as historical record (shows quality gate working)
- Mark as "FAILED" in gallery
- Revert to v9 and start fresh

**Recommendation:** Keep as historical record. Gallery should show progression and failures.

## Success Criteria

✅ Baseline reset to v9 (SSIM 0.258)
✅ Quality gate hardcoded baseline fixed
✅ SSIM measurement script created
✅ Self-learning prompt updated with mandatory workflow
✅ Quality gate enforcement documented

⏳ Pending verification:
- [ ] Test SSIM measurement script
- [ ] Test quality gate with measured values
- [ ] Verify cron job LLM provider issue resolved
- [ ] Confirm next learning session follows new workflow
- [ ] Verify degradation prevention works

## Next Learning Session Target

**Goal:** Beat v9 baseline (SSIM 0.258, Quality 6/10)

**Priority Improvements:**
1. Adaptive Line Weight - Thicker lines in dark areas
2. Better Edge Detection - Sobel + Canny combination
3. Importance Map - Edge-based scoring boost

**Success Metric:** SSIM ≥ 0.260 (≥0.8% improvement) OR Quality ≥ 7/10 with SSIM stable

**Failure Threshold:** SSIM < 0.255 (-1.2% degradation) → ROLLBACK

## Lessons Learned

1. **Never trust claims** - Always measure actual metrics
2. **Objective > Subjective** - SSIM beats quality scores
3. **Quality gate is mandatory** - No bypass, no exceptions
4. **Baseline is sacred** - Protect the known-good version
5. **Model variance is real** - Different models = different quality
6. **Degradation is silent** - Without quality gate, it goes unnoticed

## User Feedback Integration

**User frustration:** "Gamau terus degrading kayak gini. Terutama ketika ganti model AI."

**Response:** Quality gate now enforces objective metrics. Model changes won't cause degradation because quality gate measures ACTUAL output, not code quality. If new model produces worse SSIM, quality gate rejects it.

**User expectation:** "Hasilnya bagus ketika pake opus yg mahal saja."

**Response:** Quality gate is model-agnostic. If Opus produces SSIM 0.280 and Sonnet produces SSIM 0.240, quality gate will reject Sonnet's output. Baseline (v9, SSIM 0.258) is the minimum acceptable quality.

---

**Status:** READY FOR NEXT LEARNING SESSION

**Baseline:** v9.0 (SSIM 0.258, Quality 6/10) - PROTECTED

**Quality Gate:** ENFORCED - No bypass allowed

**Next Run:** Cron schedule */5 * * * * (every 5 minutes when idle)
