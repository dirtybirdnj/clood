#!/bin/bash
# Issue Catfight Processor - Universal Edition
# Runs all open issues through model gauntlet, posts analysis as comments
# Uses the Python processor for label-aware, resume-capable operation
#
# Usage:
#   ./issue_catfight.sh              # Use all available models
#   ./issue_catfight.sh --fast       # Use only fast, reliable models
#   MODELS="m1,m2" ./issue_catfight.sh  # Use specific models

set -e

REPO="dirtybirdnj/clood"
HOSTNAME=$(hostname | sed 's/.local$//')

# Fast models - proven fast & reliable from overnight benchmarking
# These all completed <40s avg with 0 failures
FAST_MODELS="qwen2.5-coder:3b,llama3.1:8b,llama3-groq-tool-use:8b,tinyllama"

# Parse command line args
FAST_MODE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --fast|-f)
            FAST_MODE=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Detect available models or use defaults
# This can be overridden with MODELS env var or --fast flag
if [ -z "$MODELS" ]; then
    if [ "$FAST_MODE" = true ]; then
        # Fast mode: use only proven fast models
        MODELS="$FAST_MODELS"
        echo "⚡ FAST MODE: Using proven fast models only"
    elif command -v ollama &> /dev/null; then
        # Try to detect models from ollama
        AVAILABLE=$(ollama list 2>/dev/null | tail -n +2 | awk '{print $1}' | tr '\n' ',' | sed 's/,$//')
        if [ -n "$AVAILABLE" ]; then
            MODELS="$AVAILABLE"
        else
            # Fallback defaults
            MODELS="qwen2.5-coder:3b,llama3.1:8b,tinyllama"
        fi
    else
        MODELS="qwen2.5-coder:3b,llama3.1:8b,tinyllama"
    fi
fi

LOG_DIR="/tmp/catfight-triage-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$LOG_DIR"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Find clood binary
if [ -x "$SCRIPT_DIR/../../clood" ]; then
    CLOOD_PATH="$SCRIPT_DIR/../../clood"
elif [ -x "$HOME/bin/clood" ]; then
    CLOOD_PATH="$HOME/bin/clood"
elif command -v clood &> /dev/null; then
    CLOOD_PATH="$(which clood)"
else
    echo "ERROR: Cannot find clood binary"
    exit 1
fi

echo "🐱 ISSUE CATFIGHT PROCESSOR"
echo "==========================="
echo "Host: $HOSTNAME"
echo "Repo: $REPO"
echo "Models: $MODELS"
echo "Clood: $CLOOD_PATH"
echo "Log dir: $LOG_DIR"
echo ""

# Get all open issues (including labels for skip logic)
ISSUES_FILE="$LOG_DIR/issues.json"
gh issue list --repo "$REPO" --state open --limit 100 --json number,title,body,labels > "$ISSUES_FILE"

COUNT=$(python3 -c "import json; print(len(json.load(open('$ISSUES_FILE'))))")
echo "Processing $COUNT issues..."
echo ""

# Export environment for Python processor
export ISSUES_FILE
export LOG_DIR
export MODELS
export REPO
export HOSTNAME
export CLOOD_PATH

# Run the label-aware processor
python3 "$SCRIPT_DIR/issue_catfight_processor.py"
