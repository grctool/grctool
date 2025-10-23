#!/bin/bash
# migrate-portable-data.sh - One-time migration to portable data structure

set -e

echo "🔄 Migrating to portable data directory structure..."

# Ensure we're in the grctool directory
if [ ! -f "go.mod" ] || [ ! -f ".grctool.yaml" ]; then
    echo "❌ Error: Run this script from the grctool directory"
    exit 1
fi

# Get data_dir from config or environment variable
DATA_DIR="${GRCTOOL_DATA_DIR:-../}"
echo "📁 Using data directory: $DATA_DIR"

# Resolve absolute path
if [[ "$DATA_DIR" = /* ]]; then
    # Already absolute
    ABS_DATA_DIR="$DATA_DIR"
else
    # Make relative path absolute
    ABS_DATA_DIR="$(cd "$DATA_DIR" && pwd)"
fi

echo "📁 Absolute data directory: $ABS_DATA_DIR"

# Create required directory structure
echo "📂 Creating directory structure..."
mkdir -p "$DATA_DIR"/{docs,evidence}
mkdir -p "$DATA_DIR/docs"/{controls,policies,evidence_tasks,evidence_prompts}
mkdir -p "$DATA_DIR/evidence"/{terraform,github,submissions}
mkdir -p "$DATA_DIR/.cache"/{prompts,summaries,tool_outputs,relationships,validations}

# Ensure .gitignore exists and includes cache
GITIGNORE="$DATA_DIR/.gitignore"
echo "📝 Managing .gitignore at $GITIGNORE"

if [ ! -f "$GITIGNORE" ]; then
    echo "# Security program data" > "$GITIGNORE"
    echo "Created new .gitignore"
fi

# Check if .cache/ is already ignored
if ! grep -q "^\.cache/" "$GITIGNORE" 2>/dev/null; then
    echo "" >> "$GITIGNORE"
    echo "# Tool cache directory" >> "$GITIGNORE"
    echo ".cache/" >> "$GITIGNORE"
    echo "✅ Added .cache/ to .gitignore"
else
    echo "✅ .cache/ already in .gitignore"
fi

# Migration of existing data
echo "📦 Migrating existing data..."

# Move existing data directory if it exists
if [ -d "data" ] && [ "$DATA_DIR" != "data" ]; then
    echo "📦 Moving existing data directory..."
    if [ -d "data/controls" ]; then
        cp -r data/controls/* "$DATA_DIR/docs/controls/" 2>/dev/null || true
        echo "✅ Migrated controls"
    fi
    if [ -d "data/policies" ]; then
        cp -r data/policies/* "$DATA_DIR/docs/policies/" 2>/dev/null || true
        echo "✅ Migrated policies"
    fi
    if [ -d "data/evidence_tasks" ]; then
        cp -r data/evidence_tasks/* "$DATA_DIR/docs/evidence_tasks/" 2>/dev/null || true
        echo "✅ Migrated evidence tasks"
    fi
    echo "📝 NOTE: Old data/ directory preserved for safety"
fi

# Move any existing docs/prompts to cache
if [ -d "../docs/prompts" ] && [ "$DATA_DIR" == "../" ]; then
    echo "📦 Moving generated prompts to cache..."
    mv ../docs/prompts/* "$DATA_DIR/.cache/prompts/" 2>/dev/null || true
    rmdir ../docs/prompts 2>/dev/null || true
    echo "✅ Moved generated prompts to cache"
fi

# Move any existing cache at isms level
if [ -d "../.cache" ] && [ "$DATA_DIR" != "../" ]; then
    echo "📦 Moving existing cache..."
    cp -r ../.cache/* "$DATA_DIR/.cache/" 2>/dev/null || true
    echo "✅ Moved cache to data directory"
fi

# Create bin directory and build
echo "🔨 Setting up build structure..."
mkdir -p bin

# Remove old binary if it exists
if [ -f "grctool" ]; then
    echo "🗑️  Removing old binary at root"
    rm grctool
fi

# Build new binary
echo "🔨 Building grctool..."
if go build -o bin/grctool; then
    echo "✅ Built grctool to bin/grctool"
else
    echo "❌ Build failed"
    exit 1
fi

# Test the new configuration
echo "🧪 Testing new configuration..."
if ./bin/grctool --help > /dev/null 2>&1; then
    echo "✅ Tool runs correctly"
else
    echo "❌ Tool execution failed"
    exit 1
fi

# Update test workflow script if it exists
if [ -f "test_workflow.sh" ]; then
    echo "🔧 Updating test workflow script..."
    sed -i.bak 's|./grctool|./bin/grctool|g' test_workflow.sh
    echo "✅ Updated test_workflow.sh (backup: test_workflow.sh.bak)"
fi

echo ""
echo "🎉 Migration complete!"
echo ""
echo "📋 Summary:"
echo "  • Data directory: $ABS_DATA_DIR"
echo "  • Cache location: $ABS_DATA_DIR/.cache/"
echo "  • Binary location: $(pwd)/bin/grctool"
echo "  • Launch Claude Code from: $(pwd)"
echo ""
echo "📝 Next steps:"
echo "  1. Test with: ./bin/grctool tool list"
echo "  2. Launch Claude Code from this directory (grctool/)"
echo "  3. Use ./bin/grctool for all commands"
echo ""
echo "🔄 To use a different data directory:"
echo "  export GRCTOOL_DATA_DIR=/path/to/your/data"
echo "  ./bin/grctool sync"