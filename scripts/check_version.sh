#!/bin/bash
# Script to check if you're using a retracted version of go-dotignore

set -e

echo "🔍 Checking go-dotignore version..."
echo ""

# Check if go-dotignore is in go.mod
if ! grep -q "github.com/codeglyph/go-dotignore" go.mod 2>/dev/null; then
    echo "✅ go-dotignore not found in go.mod (not using this package)"
    exit 0
fi

# Get current version
CURRENT_VERSION=$(go list -m github.com/codeglyph/go-dotignore 2>/dev/null | awk '{print $2}')

if [ -z "$CURRENT_VERSION" ]; then
    echo "⚠️  Could not determine current version"
    exit 1
fi

echo "📦 Current version: $CURRENT_VERSION"
echo ""

# Check if it's a retracted version
if [[ "$CURRENT_VERSION" =~ ^v1\.(0|1)\. ]]; then
    echo "❌ CRITICAL: You are using a RETRACTED version!"
    echo ""
    echo "   Versions v1.0.0-v1.1.1 contain critical bugs:"
    echo "   • Root-relative patterns (/pattern) don't work"
    echo "   • Substring matching causes false positives"
    echo "   • No escaped negation support"
    echo ""
    echo "🚀 Upgrade now:"
    echo "   go get github.com/codeglyph/go-dotignore@v2.0.0"
    echo "   go mod tidy"
    echo ""
    echo "📖 See migration guide:"
    echo "   https://github.com/codeglyph/go-dotignore/blob/main/MIGRATION.md"
    echo ""
    exit 1
elif [[ "$CURRENT_VERSION" =~ ^v2\. ]]; then
    echo "✅ You are using a supported version (v2.x)"
    echo ""

    # Check if it's the latest
    LATEST_VERSION=$(go list -m -versions github.com/codeglyph/go-dotignore 2>/dev/null | awk '{print $NF}')

    if [ "$CURRENT_VERSION" != "$LATEST_VERSION" ]; then
        echo "💡 Note: A newer version is available: $LATEST_VERSION"
        echo "   Upgrade with: go get github.com/codeglyph/go-dotignore@$LATEST_VERSION"
    else
        echo "🎉 You are on the latest version!"
    fi
    echo ""
    exit 0
else
    echo "⚠️  Unknown version format: $CURRENT_VERSION"
    exit 1
fi
