#!/bin/bash
set -e

echo "=== Cloudflare Pages Setup ==="
echo ""
echo "Step 1: Create API Token"
echo "------------------------"
echo "1. Open: https://dash.cloudflare.com/profile/api-tokens"
echo "2. Click 'Create Token' → 'Get started' (Custom token)"
echo "3. Token name: GitHub Actions - Pages Deploy"
echo "4. Permissions: Cloudflare Pages → Edit"
echo "5. Click 'Continue to summary' → 'Create Token'"
echo "6. COPY the token (shown only once!)"
echo ""
read -sp "Paste your Cloudflare API token here: " CF_TOKEN
echo ""
echo ""

if [ -z "$CF_TOKEN" ]; then
    echo "Error: Token cannot be empty"
    exit 1
fi

echo "Step 2: Setting up GitHub secrets..."
echo "------------------------------------"

# Set GitHub secrets
gh secret set CLOUDFLARE_API_TOKEN -b "$CF_TOKEN" -R hermes-ai-agent/string-art-generator
gh secret set CLOUDFLARE_ACCOUNT_ID -b "ed771e694c6365ea42180a5c54aadf6a" -R hermes-ai-agent/string-art-generator

echo "✓ GitHub secrets configured"
echo ""

echo "Step 3: Committing workflow..."
echo "-------------------------------"

cd ~/string-art
git add .github/workflows/deploy.yml
git commit -m "Add Cloudflare Pages deployment workflow"
git push

echo "✓ Workflow pushed to GitHub"
echo ""

echo "Step 4: Creating Cloudflare Pages project..."
echo "---------------------------------------------"
echo "Go to: https://dash.cloudflare.com/ed771e694c6365ea42180a5c54aadf6a/pages/new"
echo "1. Connect to Git → Select 'hermes-ai-agent/string-art-generator'"
echo "2. Build settings:"
echo "   - Framework preset: None"
echo "   - Build command: (leave empty)"
echo "   - Build output directory: docs"
echo "3. Click 'Save and Deploy'"
echo ""
echo "After first deploy, GitHub Actions will handle future deployments automatically."
echo ""
echo "=== Setup Complete ==="
