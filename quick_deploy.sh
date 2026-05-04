#!/bin/bash
# Quick deploy to Cloudflare Pages using API token
# Usage: CLOUDFLARE_API_TOKEN=your_token ./quick_deploy.sh

set -e

if [ -z "$CLOUDFLARE_API_TOKEN" ]; then
    echo "❌ Error: CLOUDFLARE_API_TOKEN not set"
    echo ""
    echo "Usage:"
    echo "  export CLOUDFLARE_API_TOKEN=your_token"
    echo "  ./quick_deploy.sh"
    echo ""
    echo "Or:"
    echo "  CLOUDFLARE_API_TOKEN=your_token ./quick_deploy.sh"
    exit 1
fi

echo "🚀 Deploying to Cloudflare Pages..."
echo ""

# Set token for wrangler
export CLOUDFLARE_API_TOKEN="$CLOUDFLARE_API_TOKEN"

# Add wrangler to PATH
export PATH="$HOME/.hermes/node/bin:$PATH"

# Deploy
cd ~/string-art
wrangler pages deploy docs \
    --project-name=string-art-generator \
    --branch=main

echo ""
echo "✅ Deployment complete!"
echo "🔗 Gallery: https://string-art-generator.pages.dev"
