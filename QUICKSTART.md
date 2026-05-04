# String Art Gallery - Quick Start

## 🚀 Deploy Gallery ke Cloudflare Pages

### 1. Install Wrangler (One-time)

```bash
npm install -g wrangler
```

### 2. Login ke Cloudflare (One-time)

```bash
wrangler login
```

### 3. Deploy Gallery

```bash
cd ~/string-art
./deploy_gallery.sh
```

**Done!** Gallery live di: https://string-art-generator.pages.dev

---

## 📊 Update Gallery dengan Versi Baru

### Manual Generation

```bash
# 1. Generate string art
./string-art-gen --input photo.jpg --output output/result.svg --pins 300 --lines 2500 --wu

# 2. Update gallery + auto-deploy
./update_gallery.py output/result.svg \
  --version v12 \
  --ssim 0.234 \
  --quality 8 \
  --pins 300 \
  --lines 2500 \
  --description "Manual: Wu anti-aliased" \
  --deploy
```

### Self-Learning (Fully Automated)

```bash
# Run self-learning (auto-updates gallery)
./self_learning_v2.py
```

Untuk auto-deploy setiap run, edit `self_learning_v2.py` dan tambahkan `--deploy` flag.

---

## 🔄 Regenerate Manifest

```bash
# Scan all files in docs/ and rebuild manifest
./generate_manifest.py
```

---

## 📁 File Structure

```
string-art/
├── docs/                     # Gallery files (deployed to Cloudflare)
│   ├── gallery.html          # Main gallery page
│   ├── results_manifest.json # Gallery data (29 versions)
│   └── result_v*.svg/png     # Generated files
├── deploy_gallery.sh         # Deploy to Cloudflare Pages
├── update_gallery.py         # Update gallery with new version
├── generate_manifest.py      # Regenerate manifest
├── self_learning_v2.py       # Self-learning script
└── string-art-gen            # Go binary
```

---

## 🎯 Aturan String Art (WAJIB)

- ✅ 360 pins max (1 pin per degree)
- ✅ 0.18mm stroke width, full opaque
- ✅ SVG 600mm x 600mm
- ✅ Raster hanya preview

---

## 📖 Documentation

- **Gallery System:** [GALLERY_SYSTEM.md](GALLERY_SYSTEM.md)
- **Cloudflare Deploy:** [CLOUDFLARE_DEPLOY.md](CLOUDFLARE_DEPLOY.md)
- **Main README:** [README.md](README.md)

---

## 🏆 Current Status

**Baseline:** v10.0 (SSIM: 0.1959)  
**Best Ever:** v9 Birsak (SSIM: 0.2580)  
**Total Versions:** 29  
**Gallery URL:** https://string-art-generator.pages.dev
