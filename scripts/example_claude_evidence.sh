#!/bin/bash
# Example script for generating evidence with Claude AI

# Set your Claude API key
export CLAUDE_API_KEY="your-claude-api-key-here"

# Tugboat auth is handled via browser login

# Change to grctool directory
cd "$(dirname "$0")/.." || exit 1

# Build the tool if not already built
if [ ! -f "build/grctool" ]; then
    echo "Building grctool..."
    make build
fi

echo "🚀 Claude AI Evidence Generation Example"
echo "========================================"
echo ""

# Step 1: Sync evidence tasks from Tugboat
echo "📥 Step 1: Syncing evidence tasks..."
./build/grctool sync --evidence
echo ""

# Step 2: List evidence tasks to see what's available
echo "📋 Step 2: Listing evidence tasks..."
./build/grctool evidence list --framework SOC2 --status pending | head -20
echo ""

# Step 3: Analyze a specific task (example task ID)
TASK_ID=328123  # Replace with your actual task ID
echo "🔍 Step 3: Analyzing task $TASK_ID..."
./build/grctool evidence analyze $TASK_ID
echo ""

# Step 4: Generate evidence using Claude and tools
echo "🤖 Step 4: Generating evidence with Claude AI..."
echo "Using tools: terraform, github"
echo ""

# Generate evidence with Terraform scanning
./build/grctool evidence generate $TASK_ID \
    --tools terraform \
    --format csv \
    --output evidence/generated/

echo ""
echo "📂 Evidence saved to: evidence/generated/"
echo ""

# Step 5: Review the generated evidence
echo "👀 Step 5: Reviewing generated evidence..."
./build/grctool evidence review $TASK_ID --show-reasoning
echo ""

# Example: Batch generation for multiple tasks
echo "📦 Example: Batch generation for all pending SOC2 tasks"
echo "To generate evidence for all pending tasks, you would run:"
echo ""
echo "  ./build/grctool evidence generate --all --tools terraform,github --format csv"
echo ""

# Example: Generate with specific tools based on task type
echo "🛠️  Example: Tool-specific generation"
echo ""
echo "For access control tasks:"
echo "  ./build/grctool evidence generate $TASK_ID --tools terraform --format markdown"
echo ""
echo "For policy documentation:"
echo "  ./build/grctool evidence generate $TASK_ID --tools github --format markdown"
echo ""

echo "✅ Example complete!"
echo ""
echo "Next steps:"
echo "1. Review the generated evidence in evidence/generated/"
echo "2. Make any necessary adjustments"
echo "3. Submit the evidence: ./build/grctool evidence submit $TASK_ID"
echo ""
echo "⚠️  Remember to:"
echo "- Set your CLAUDE_API_KEY environment variable"
echo "- Configure tool settings in .grctool.yaml"
echo "- Review all generated evidence before submission"