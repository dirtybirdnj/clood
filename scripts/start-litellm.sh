#!/bin/bash
#
# start-litellm.sh - Start LiteLLM proxy for multi-machine Ollama access
#
# Usage: ./scripts/start-litellm.sh
#        ./scripts/start-litellm.sh --background
#

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/../infrastructure/litellm-config.yaml"
VENV_PATH="$HOME/.local/share/litellm-venv"
PORT=4000

# Check venv exists
if [ ! -f "$VENV_PATH/bin/litellm" ]; then
    echo "LiteLLM not installed. Installing..."
    python3 -m venv "$VENV_PATH"
    "$VENV_PATH/bin/pip" install 'litellm[proxy]' --quiet
fi

# Check config exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Config not found at $CONFIG_FILE"
    exit 1
fi

echo "Starting LiteLLM proxy on http://localhost:$PORT"
echo "Config: $CONFIG_FILE"
echo ""
echo "Available models:"
grep "model_name:" "$CONFIG_FILE" | sed 's/.*model_name: "/  • /;s/"//'
echo ""

if [ "$1" = "--background" ]; then
    echo "Running in background. Logs: /tmp/litellm.log"
    nohup "$VENV_PATH/bin/litellm" --config "$CONFIG_FILE" --port $PORT > /tmp/litellm.log 2>&1 &
    echo "PID: $!"
else
    echo "Press Ctrl+C to stop"
    echo ""
    "$VENV_PATH/bin/litellm" --config "$CONFIG_FILE" --port $PORT
fi
