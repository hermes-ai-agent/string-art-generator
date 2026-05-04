#!/bin/bash
# Deploy string art gallery to Cloudflare Pages
# Usage: ./deploy_gallery.sh [--auto]

set -e

PROJECT_NAME="string-art-generator"
ACCOUNT_ID="ed771e694c6365ea42180a5c54aadf6a"
DOCS_DIR="docs"

echo "🚀 Deploying String Art Gallery to Cloudflare Pages"
echo ""

# Check if wrangler is installed
if ! command -v wrangler &> /dev/null; then
    echo "⚠️  Wrangler not found. Installing..."
    npm install -g wrangler
fi

# Check if logged in
if ! wrangler whoami &> /dev/null; then
    echo "🔐 Please login to Cloudflare:"
    wrangler login
fi

# Regenerate manifest before deploy
echo "📊 Regenerating manifest..."
cd ~/string-art
python3 generate_manifest.py

# Check if there are changes
if [[ "$1" != "--auto" ]]; then
    echo ""
    echo "📁 Files to deploy:"
    ls -lh $DOCS_DIR/*.html $DOCS_DIR/*.json 2>/dev/null | tail -5
    echo ""
    read -p "Continue with deployment? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ Deployment cancelled"
        exit 1
    fi
fi

# Deploy to Cloudflare Pages
echo ""
echo "🌐 Deploying to Cloudflare Pages..."
wrangler pages deploy $DOCS_DIR \
    --project-name=$PROJECT_NAME \
    --branch=main

echo ""
echo "✅ Deployment complete!"
echo ""
echo "🔗 Gallery URL: https://string-art-generator.pages.dev"
echo "📊 Dashboard: https://dash.cloudflare.com/$ACCOUNT_ID/pages/view/$PROJECT_NAME"
