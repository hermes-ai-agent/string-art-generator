# Self-Learning Gallery

Gallery page untuk menampilkan semua hasil self-learning dengan lazy load dan quality metrics.

## Features

### 📊 Stats Summary
- **Total Results** - Jumlah total hasil self-learning
- **Best SSIM** - SSIM tertinggi yang pernah dicapai
- **Avg Time** - Rata-rata generation time
- **Success Rate** - Persentase improvement yang berhasil (tidak failed)

### 🎨 Gallery Grid
- **Lazy Load** - Images dimuat hanya saat visible (infinite scroll ready)
- **Quality Badges** - Color-coded SSIM badges:
  - 🟢 Excellent: SSIM ≥ 0.25
  - 🔵 Good: SSIM ≥ 0.20
  - 🟡 Fair: SSIM ≥ 0.15
  - 🔴 Poor: SSIM < 0.15
- **Version Badges** - Status indicators:
  - 🟢 Best: Highest SSIM
  - 🔴 Failed: Quality gate rejected
  - 🔵 Normal: Successful improvement

### 🔍 Filters
- **All** - Tampilkan semua hasil
- **Excellent** - Hanya SSIM ≥ 0.25
- **Good** - Hanya SSIM ≥ 0.20
- **Failed** - Hanya yang failed quality gate

### 🖼️ Image Modal
- Click pada image untuk zoom full screen
- View SVG (zoomable vector)
- View PNG (raster preview)

## Data Source

Gallery menggunakan `results_manifest.json` yang di-generate otomatis oleh `generate_manifest.py`.

### Manifest Format

```json
[
  {
    "version": "v9",
    "description": "Birsak Supersampling + Enhanced Scoring",
    "ssim": 0.258,
    "quality": 6,
    "time": 11.4,
    "pins": 300,
    "lines": 2500,
    "failed": false,
    "timestamp": "2026-05-03T10:14:00",
    "svg": "test_v9_birsak.svg",
    "png": "test_v9_birsak_mobile_400px.png"
  }
]
```

## Auto-Update Workflow

Self-learning cron job otomatis update gallery:

1. ✅ Quality gate passed
2. ✅ Upload SVG+PNG ke docs/
3. ✅ Update website (update_website.py)
4. ✅ **Update gallery manifest (generate_manifest.py)** ← NEW
5. ✅ Commit and push
6. ✅ GitHub Actions deploy
7. ✅ Gallery otomatis update dengan hasil terbaru

## Usage

### View Gallery

**Production:**
https://string-art-generator-2ih.pages.dev/gallery.html

**Local:**
```bash
cd ~/string-art/docs
python3 -m http.server 8000
# Open http://localhost:8000/gallery.html
```

### Update Manifest

```bash
cd ~/string-art
python3 generate_manifest.py
```

Output:
```
✅ Generated manifest with 3 results
   Saved to: /home/amin/string-art/docs/results_manifest.json

📊 Summary:
   Total: 3
   Successful: 2
   Failed: 1
   Best SSIM: 0.2580
```

### Add New Result Manually

Edit `docs/results_manifest.json`:

```json
{
  "version": "v32",
  "description": "Your improvement description",
  "ssim": 0.265,
  "quality": 7,
  "time": 10.5,
  "pins": 300,
  "lines": 2500,
  "failed": false,
  "timestamp": "2026-05-03T19:00:00",
  "svg": "result_v32_20260503_190000.svg",
  "png": "result_v32_20260503_190000.png"
}
```

Commit dan push untuk deploy.

## Performance

### Lazy Loading
- Images dimuat hanya saat scroll ke viewport
- Menggunakan `IntersectionObserver` API
- Smooth fade-in animation
- Minimal initial load time

### Infinite Scroll Ready
- Gallery bisa menampilkan ratusan hasil tanpa lag
- Lazy load memastikan hanya visible images yang dimuat
- Memory efficient

## Monitoring

### Check Stats
```bash
curl -s https://string-art-generator-2ih.pages.dev/results_manifest.json | jq '.[0]'
```

### Count Results
```bash
curl -s https://string-art-generator-2ih.pages.dev/results_manifest.json | jq 'length'
```

### Find Best SSIM
```bash
curl -s https://string-art-generator-2ih.pages.dev/results_manifest.json | jq 'max_by(.ssim)'
```

## Future Enhancements

1. **Search** - Search by description, version, or metrics
2. **Sort** - Sort by SSIM, time, quality, or date
3. **Compare** - Side-by-side comparison of 2 versions
4. **Download** - Bulk download all results
5. **Charts** - Progress charts (SSIM over time, quality trend)
6. **Pagination** - Load more button for very large datasets
7. **Export** - Export manifest as CSV or Excel

## Summary

**Before Gallery:**
- Hasil self-learning tersebar di docs/ folder ❌
- Sulit track progress ❌
- Tidak ada visual comparison ❌

**After Gallery:**
- Semua hasil terpusat di satu page ✅
- Stats summary untuk quick overview ✅
- Lazy load untuk performance ✅
- Quality badges untuk quick assessment ✅
- Filter untuk fokus pada hasil tertentu ✅
- Auto-update setiap improvement ✅

**Gallery URL:** https://string-art-generator-2ih.pages.dev/gallery.html

Sekarang kamu bisa pantau semua hasil self-learning dengan mudah! 🎉
