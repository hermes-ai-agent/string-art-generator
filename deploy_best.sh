#!/bin/bash
# Auto-deploy best string art to GitHub Pages
# Called by self-learning cron after quality validation passes
#
# Usage: ./deploy_best.sh <svg_path> <version_label> <params_description>
# Example: ./deploy_best.sh docs/cat_v7.0_improved.svg "v7.0" "300 pins, 3000 lines, weight 28"

set -e

SVG_PATH="$1"
VERSION="$2"
PARAMS="$3"

if [ -z "$SVG_PATH" ] || [ -z "$VERSION" ] || [ -z "$PARAMS" ]; then
    echo "Usage: $0 <svg_path> <version_label> <params_description>"
    exit 1
fi

if [ ! -f "$SVG_PATH" ]; then
    echo "ERROR: SVG file not found: $SVG_PATH"
    exit 1
fi

cd /home/amin/string-art

SVG_FILENAME=$(basename "$SVG_PATH")

# Update index.html - replace the main showcase SVG
sed -i "s|<img src=\"cat_v[^\"]*\.svg\" alt=\"String Art v[^\"]*\">|<img src=\"$SVG_FILENAME\" alt=\"String Art $VERSION\">|" docs/index.html
sed -i "s|SVG Output v[0-9.]* (Zoomable) - [0-9]* pins|SVG Output $VERSION (Zoomable)|" docs/index.html
sed -i "s|[0-9]* pins, [0-9]* lines, .*</p>|$PARAMS</p>|" docs/index.html

# Update comparison section too
sed -i "s|<img src=\"cat_v[^\"]*\.svg\" alt=\"v[^\"]*- [0-9]* pins (new)\">|<img src=\"$SVG_FILENAME\" alt=\"$VERSION (new)\">|" docs/index.html

# Git commit and push
git add -A
git add -f docs/*_canvas.png docs/*_mobile_400px.png 2>/dev/null || true
git commit -m "Auto-deploy $VERSION: $PARAMS" || true
git push origin main 2>&1

echo "✅ Deployed $VERSION to GitHub Pages"
echo "🔗 https://hermes-ai-agent.github.io/string-art-generator/"
