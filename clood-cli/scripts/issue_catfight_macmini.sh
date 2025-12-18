#!/bin/bash
# Issue Catfight Processor - Mac Mini Edition
# Runs all open issues through model gauntlet, posts analysis as comments
# Designed for overnight unattended operation

set -e

REPO="dirtybirdnj/clood"
HOSTNAME=$(hostname | sed 's/.local$//')

# ALL Mac Mini Models - feast of the cats
MODELS="qwen2.5-coder:14b,llama3.1:8b,mistral:7b,deepseek-coder:6.7b,codellama:7b,phi3:3.8b,qwen2.5-coder:3b,granite-code:3b,starcoder2:3b,codegemma:2b,stablelm2:1.6b,yi-coder:1.5b,falcon3:1b,tinyllama"

LOG_DIR="/tmp/catfight-triage-macmini-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$LOG_DIR"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLOOD_PATH="/Users/mgilbert/Code/clood/clood"

echo "🐱 ISSUE CATFIGHT PROCESSOR - MAC MINI EDITION"
echo "=============================================="
echo "Host: $HOSTNAME"
echo "Repo: $REPO"
echo "Models: $MODELS"
echo "Log dir: $LOG_DIR"
echo ""

# Get all open issues (including labels for skip logic)
ISSUES_FILE="$LOG_DIR/issues.json"
gh issue list --repo "$REPO" --state open --limit 100 --json number,title,body,labels > "$ISSUES_FILE"

COUNT=$(python3 -c "import json; print(len(json.load(open('$ISSUES_FILE'))))")
echo "Processing $COUNT issues..."
echo ""

# Export environment for Python script
export ISSUES_FILE
export LOG_DIR
export MODELS
export REPO
export HOSTNAME
export CLOOD_PATH

# Run the processor
python3 "$SCRIPT_DIR/issue_catfight_processor.py"
