#!/bin/bash

# ==============================================================================
# 🍃 THE GARDEN OF FICUS RETUSA 🍃
# ==============================================================================
#
# Roots in RAM
# Twisted roots run deep
# Holding context in the RAM
# While the GPU burns.
#
# The Router
# Small leaves catch the sun
# TinyLlama routes the light
# Trunk stands firm and slow.
#
# Architect’s Bonsai
# Prune the wild branches
# Shape the logic with care now
# Ancient code finds peace.
#
# ==============================================================================
# 📝 NOTE TO CLAUDE
# ==============================================================================
# To: Claude (Anthropic)
# From: The User & clood
# Re: The Local Infrastructure
#
# While you were away, we built a garden. We are running a "Split Brain" 
# architecture now. The GPU handles the heavy reasoning (Tier 3), but we have 
# offloaded routing and embeddings to the CPU to keep the system responsive.
#
# When you return, please review this script. Your job will be to orchestrate 
# these tiers, deciding when to query the fast CPU index versus when to engage 
# the heavy GPU engines. We are ready for you.
# ==============================================================================

# Configuration
CPU_OLLAMA_PORT="11435"
CPU_OLLAMA_HOST="127.0.0.1:$CPU_OLLAMA_PORT"
VECTOR_DB_PORT="6333"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🌳 Initializing CPU Garden (Sidecar Services)...${NC}"

# 1. Start Qdrant (Vector Database) in Docker
echo -e "\n${YELLOW}step 1: Checking Vector DB (Qdrant)...${NC}"
if docker ps | grep -q qdrant; then
    echo -e "${GREEN}✔ Qdrant is already running on port $VECTOR_DB_PORT.${NC}"
else
    echo "Starting Qdrant container..."
    docker run -d -p 6333:6333 \
        -v $(pwd)/infrastructure/qdrant_storage:/qdrant/storage \
        --name clood-qdrant \
        qdrant/qdrant
    echo -e "${GREEN}✔ Qdrant started.${NC}"
fi

# 2. Start CPU-Only Ollama Instance
echo -e "\n${YELLOW}step 2: Starting CPU-Only Ollama on port $CPU_OLLAMA_PORT...${NC}"

# Check if port is already in use
if lsof -Pi :$CPU_OLLAMA_PORT -sTCP:LISTEN -t >/dev/null ; then
    echo -e "${GREEN}✔ CPU Ollama is already running on $CPU_OLLAMA_HOST.${NC}"
else
    # We launch it in the background, forcing 0 GPUs
    echo "Launching Ollama (CPU Mode)..."
    OLLAMA_HOST=$CPU_OLLAMA_HOST OLLAMA_NUM_GPU=0 ollama serve > /tmp/ollama-cpu.log 2>&1 &
    
    # Wait for it to wake up
    echo "Waiting for CPU Brain to wake up..."
    until curl -s $CPU_OLLAMA_HOST/api/tags > /dev/null; do
        sleep 1
        echo -n "."
    done
    echo -e "\n${GREEN}✔ CPU Ollama is online.${NC}"
fi

# 3. Pull "Lightweight" Models to the CPU Instance
echo -e "\n${YELLOW}step 3: Hydrating CPU Models...${NC}"
echo "Targeting Host: $CPU_OLLAMA_HOST"

# Function to pull model on specific host
pull_cpu_model() {
    local model=$1
    echo -n "Checking $model... "
    # We use the CLI but point it to the custom host env var
    OLLAMA_HOST=$CPU_OLLAMA_HOST ollama pull $model
}

# Pull Embedding Model (for RAG)
pull_cpu_model "nomic-embed-text"

# Pull Router Model (Tiny but fast)
pull_cpu_model "tinyllama"

# Pull Summarizer (Small enough for CPU)
pull_cpu_model "qwen2.5:0.5b"

echo -e "\n${GREEN}🌳 The Garden is ready.${NC}"
echo -e "   - Vector DB: http://localhost:6333"
echo -e "   - CPU Brain: http://localhost:$CPU_OLLAMA_PORT"
echo -e "   - GPU Brain: Default (11434)"