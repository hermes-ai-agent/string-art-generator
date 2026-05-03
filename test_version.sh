#!/bin/bash
# Quick test script for string art generator versions

set -e

VERSION=${1:-v26}
PINS=${2:-300}
LINES=${3:-3000}
WEIGHT=${4:-28}
MIN_DIST=${5:-15}
EDGE_WEIGHT=${6:-2.0}

echo "==================================="
echo "String Art Generator Test"
echo "==================================="
echo "Version: $VERSION"
echo "Pins: $PINS"
echo "Lines: $LINES"
echo "Weight: $WEIGHT"
echo "Min Distance: $MIN_DIST"
echo "Edge Weight: $EDGE_WEIGHT"
echo "==================================="
echo ""

# Build
echo "Building..."
go build -o string-art-gen .

# Generate
OUTPUT="docs/test_${VERSION}.svg"
echo "Generating $OUTPUT..."
./string-art-gen \
  --input cat_photo.jpg \
  --pins $PINS \
  --lines $LINES \
  --weight $WEIGHT \
  --min-dist $MIN_DIST \
  --edge-weight $EDGE_WEIGHT \
  --output $OUTPUT \
  --$VERSION

# Validate
echo ""
echo "==================================="
echo "Quality Validation"
echo "==================================="
python3 quality_validator.py $OUTPUT cat_photo.jpg

# Show mobile preview path
MOBILE_PNG="${OUTPUT%.svg}_mobile_400px.png"
echo ""
echo "==================================="
echo "Visual Review"
echo "==================================="
echo "Mobile preview: $MOBILE_PNG"
echo ""
echo "Check the preview for:"
echo "  - Cat features clearly visible (eyes, ears, nose)"
echo "  - No solid black blobs"
echo "  - No over-empty areas"
echo "  - Good tonal balance"
echo ""
