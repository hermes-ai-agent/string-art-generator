# Cloudflare Pages Deployment

Gallery string art di-deploy ke Cloudflare Pages untuk akses publik.

## Setup (One-time)

### 1. Install Wrangler

```bash
npm install -g wrangler
```

### 2. Login ke Cloudflare

```bash
wrangler login
```

Browser akan terbuka, login dengan akun Cloudflare Anda.

### 3. Create Pages Project (First Deploy)

```bash
cd ~/string-art
wrangler pages deploy docs --project-name=string-art-generator
```

Wrangler akan:
- Create project `string-art-generator` di Cloudflare Pages
- Deploy `docs/` folder
- Generate URL: `https://string-art-generator.pages.dev`

## Usage

### Manual Deploy

```bash
# Deploy gallery to Cloudflare Pages
./deploy_gallery.sh
```

Script akan:
1. Regenerate manifest dari semua files
2. Deploy ke Cloudflare Pages
3. Show deployment URL

### Auto-Deploy with Gallery Update

```bash
# Update gallery and auto-deploy
./update_gallery.py output/result.svg \
  --version v12 \
  --ssim 0.234 \
  --pins 300 \
  --lines 2500 \
  --deploy  # ← Auto-deploy flag
```

### Self-Learning with Auto-Deploy

Edit `self_learning_v2.py` dan tambahkan `--deploy` flag:

```python
update_cmd = [
    'python3', '/home/amin/string-art/update_gallery.py',
    str(output_file),
    '--version', version,
    '--ssim', str(ssim),
    '--quality', str(min(10, int(ssim * 40))),
    '--pins', str(params['pins']),
    '--lines', str(params['lines']),
    '--description', f"Self-learning: alpha={params['alpha']}, +{improvement:.1f}% SSIM",
    '--deploy'  # ← Add this
]
```

## Deployment Info

**Project Name:** `string-art-generator`  
**Account ID:** `ed771e694c6365ea42180a5c54aadf6a`  
**Deploy Directory:** `docs/`  
**Production URL:** `https://string-art-generator.pages.dev`

## Files Deployed

```
docs/
├── gallery.html              # Main gallery page
├── index.html                # Landing page
├── self_learning.html        # Self-learning dashboard
├── svg-viewer.html           # SVG viewer
├── results_manifest.json     # Gallery data (29 versions)
├── result_v*.svg             # Generated SVGs
├── result_v*.png             # Preview PNGs
└── examples/
    └── cat_photo.jpg         # Test image
```

## Cloudflare Dashboard

- **Pages Dashboard:** https://dash.cloudflare.com/ed771e694c6365ea42180a5c54aadf6a/pages
- **Project Settings:** https://dash.cloudflare.com/ed771e694c6365ea42180a5c54aadf6a/pages/view/string-art-generator
- **Deployments:** https://dash.cloudflare.com/ed771e694c6365ea42180a5c54aadf6a/pages/view/string-art-generator/deployments

## Custom Domain (Optional)

1. Go to project settings
2. Click "Custom domains"
3. Add domain: `string-art.yourdomain.com`
4. Add CNAME record to DNS:
   ```
   string-art CNAME string-art-generator.pages.dev
   ```

## Troubleshooting

### Wrangler not found

```bash
npm install -g wrangler
```

### Not logged in

```bash
wrangler login
```

### Deployment failed

```bash
# Check wrangler status
wrangler whoami

# Manual deploy with verbose output
wrangler pages deploy docs --project-name=string-art-generator --verbose
```

### Wrong directory deployed

Check `wrangler.toml`:
```toml
pages_build_output_dir = "docs"  # Must be "docs", not "public"
```

## Automation

### Deploy on every gallery update

Add to `~/.bashrc` or create alias:

```bash
alias update-gallery='~/string-art/update_gallery.py "$@" --deploy'
```

Usage:
```bash
update-gallery output/result.svg --version v12 --ssim 0.234 --pins 300 --lines 2500
```

### Cron job for periodic deploy

```bash
# Deploy gallery every hour
0 * * * * cd ~/string-art && ./deploy_gallery.sh --auto
```

## Cost

Cloudflare Pages Free Tier:
- ✅ Unlimited requests
- ✅ Unlimited bandwidth
- ✅ 500 builds/month
- ✅ 1 build at a time

Perfect untuk gallery static site.
