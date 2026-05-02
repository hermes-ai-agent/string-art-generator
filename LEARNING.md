# String Art Learning System

Sistem pembelajaran otomatis untuk continuous improvement string art generator.

## Cara Kerja

1. **Self-learning cron job** berjalan setiap 15 menit
2. Jika idle (13+ menit tanpa activity), mulai learning session
3. Research string art algorithms, techniques, implementations
4. Evaluate current code untuk improvement opportunities
5. Implement improvements langsung (auto-apply)
6. Test implementation
7. Update documentation
8. Report via Telegram

## Topic Management

### Default Topics (3 Rotation)
- **Self** - Hermes Agent updates
- **World** - Technology news
- **Allah** - Islamic knowledge

### Override Topic (Single Focus)
- **String-Art** - String art Go development (CURRENT)
  - Focus: Go implementation at `~/string-art/*.go`
  - Target: Performance, algorithms, Go-specific optimizations
  - Status: v3.1.0 - Research improvements ported to Go

### Commands

```bash
# Check current topic
python3 ~/.hermes/scripts/topic_manager.py status

# Set override topic (string-art)
python3 ~/.hermes/scripts/topic_manager.py set string-art \
  "String art algorithms and optimization techniques" \
  "String art mathematical theory and computational geometry" \
  "Advanced string art implementations and code examples"

# Reset to default 3-topic rotation
python3 ~/.hermes/scripts/topic_manager.py reset
```

## Learning Focus Areas

### 1. Algorithms
- **Current:** Greedy line selection (Go implementation)
- **Targets:** TSP-based, genetic algorithms, simulated annealing, GPU acceleration
- **Goal:** Better image matching with fewer lines, maintain Go performance

### 2. Pin Arrangements
- **Current:** Circular only
- **Targets:** Square, hexagon, custom shapes
- **Goal:** More flexibility for different art styles

### 3. Performance
- **Current:** Go with parallel workers (8 workers, 2.5s for 2000 lines)
- **Targets:** Line caching, SIMD, GPU compute shaders, better concurrency
- **Goal:** Sub-second generation, real-time preview

### 4. Rendering
- **Current:** Basic line drawing
- **Targets:** Anti-aliasing, variable thickness, transparency
- **Goal:** More realistic preview

### 5. Multi-Color
- **Current:** Single color (black)
- **Targets:** Multiple string colors, color optimization
- **Goal:** Colorful string art

### 6. Physical Construction
- **Current:** Digital only
- **Targets:** String tension, material properties, construction order
- **Goal:** Practical physical construction guidance

## Expected Improvements

Setiap learning session (setiap ~15-30 menit saat idle):
- Research 1-2 techniques
- Evaluate applicability
- Implement if beneficial
- Test and document

**Timeline estimate:**
- Week 1: Algorithm improvements (TSP, genetic)
- Week 2: Pin arrangement variations
- Week 3: Performance optimizations
- Week 4: Rendering improvements
- Ongoing: Physical construction insights

## Monitoring

### Check Learning Logs
```bash
# List all string-art learning sessions
ls -lht ~/.hermes/learning_logs/*string-art*

# View latest
cat ~/.hermes/learning_logs/$(ls -t ~/.hermes/learning_logs/*string-art* | head -1)
```

### Check Code Changes
```bash
# View git log (if initialized)
cd ~/string-art
git log --oneline

# View current version
head -20 string_art_generator.py | grep "Version:"
```

### Check README Updates
```bash
# View learning log section
cat ~/string-art/README.md | grep -A 20 "Learning Log"
```

## Manual Testing

```bash
cd ~/string-art

# Test with simple shape
python3 string_art_generator.py test_circle.png --pins 100 --lines 500

# Test with real image
python3 string_art_generator.py your_image.jpg --pins 200 --lines 3000

# Test with different parameters
python3 string_art_generator.py your_image.jpg \
  --pins 300 \
  --lines 4000 \
  --min-distance 20 \
  --line-weight 25
```

## Rollback

Jika improvement menyebabkan masalah:

```bash
# Via git (if initialized)
cd ~/string-art
git log --oneline
git revert <commit-hash>

# Manual rollback
# Restore from backup (auto-created before changes)
cp string_art_generator.py.backup string_art_generator.py
```

## Disable Learning

```bash
# Pause self-learning cron
hermes cron pause 2b3bea51d716

# Resume
hermes cron resume 2b3bea51d716
```

## Switch Back to Default Topics

```bash
# Reset to 3-topic rotation (Self, World, Allah)
python3 ~/.hermes/scripts/topic_manager.py reset

# Verify
python3 ~/.hermes/scripts/topic_manager.py status
```

---

**Status:** Active - Learning Go string-art every ~15-30 minutes when idle
**Last Updated:** 2026-05-02
**Current Version:** v3.1.0 (Go + Research improvements)
