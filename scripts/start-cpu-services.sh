#!/bin/bash
set -e

# ==============================================================================
# THE GARDEN OF FICUS RETUSA - CPU Sidecar Services
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
# ==============================================================================

# Source common configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/clood-common.sh"

# ==============================================================================
# CONFIGURATION
# ==============================================================================

QDRANT_CONTAINER="clood-qdrant"
QDRANT_STORAGE="$CLOOD_ROOT/infrastructure/qdrant_storage"
MAX_WAIT_SECONDS=30

# ==============================================================================
# FUNCTIONS
# ==============================================================================

start_qdrant() {
    log_tier info "Checking Vector DB (Qdrant)..."

    if docker ps --format '{{.Names}}' | grep -q "^${QDRANT_CONTAINER}$"; then
        log_tier ok "Qdrant is already running on port $VECTOR_DB_PORT"
        return 0
    fi

    # Check if container exists but stopped
    if docker ps -a --format '{{.Names}}' | grep -q "^${QDRANT_CONTAINER}$"; then
        log_tier info "Starting existing Qdrant container..."
        docker start "$QDRANT_CONTAINER"
    else
        log_tier info "Creating new Qdrant container..."
        mkdir -p "$QDRANT_STORAGE"
        docker run -d \
            -p "$VECTOR_DB_PORT:6333" \
            -v "$QDRANT_STORAGE:/qdrant/storage" \
            --name "$QDRANT_CONTAINER" \
            qdrant/qdrant
    fi

    # Wait for Qdrant to be ready
    if wait_for_service "$VECTOR_DB_URL/collections" "Qdrant" "$MAX_WAIT_SECONDS"; then
        log_tier ok "Qdrant started successfully"
    else
        log_tier error "Qdrant failed to start within ${MAX_WAIT_SECONDS}s"
        return 1
    fi
}

start_cpu_ollama() {
    log_tier info "Checking CPU-Only Ollama on port $CPU_OLLAMA_PORT..."

    if lsof -Pi :"$CPU_OLLAMA_PORT" -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_tier ok "CPU Ollama is already running on $CPU_OLLAMA_HOST"
        return 0
    fi

    log_tier info "Launching Ollama (CPU Mode)..."

    # Launch with 0 GPUs to force CPU-only mode
    OLLAMA_HOST="$CPU_OLLAMA_HOST" OLLAMA_NUM_GPU=0 ollama serve > /tmp/ollama-cpu.log 2>&1 &
    local ollama_pid=$!

    # Give it a moment to crash if it's going to
    sleep 2
    if ! kill -0 "$ollama_pid" 2>/dev/null; then
        log_tier error "CPU Ollama failed to start. Check /tmp/ollama-cpu.log"
        return 1
    fi

    # Wait for API to respond
    if wait_for_service "$CPU_OLLAMA_URL/api/tags" "CPU Ollama" "$MAX_WAIT_SECONDS"; then
        log_tier ok "CPU Ollama is online (PID: $ollama_pid)"
    else
        log_tier error "CPU Ollama failed to respond within ${MAX_WAIT_SECONDS}s"
        kill "$ollama_pid" 2>/dev/null || true
        return 1
    fi
}

pull_cpu_models() {
    log_tier info "Hydrating CPU Models..."

    local models=("$MODEL_ROUTER" "$MODEL_EMBED" "$MODEL_SUMMARIZE")

    for model in "${models[@]}"; do
        log_tier cpu "Pulling $model..."
        if OLLAMA_HOST="$CPU_OLLAMA_HOST" ollama pull "$model"; then
            log_tier ok "$model ready"
        else
            log_tier warn "Failed to pull $model (may already exist or network issue)"
        fi
    done
}

# ==============================================================================
# MAIN
# ==============================================================================

main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Initializing CPU Garden (Sidecars)   ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # Step 1: Start Qdrant
    echo -e "${YELLOW}Step 1/3: Vector Database${NC}"
    if ! start_qdrant; then
        log_tier error "Failed to start Qdrant. Aborting."
        exit 1
    fi
    echo ""

    # Step 2: Start CPU Ollama
    echo -e "${YELLOW}Step 2/3: CPU Ollama Instance${NC}"
    if ! start_cpu_ollama; then
        log_tier error "Failed to start CPU Ollama. Aborting."
        exit 1
    fi
    echo ""

    # Step 3: Pull Models
    echo -e "${YELLOW}Step 3/3: Model Hydration${NC}"
    pull_cpu_models
    echo ""

    # Summary
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  The Garden is Ready                  ${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo -e "   Vector DB:  $VECTOR_DB_URL"
    echo -e "   CPU Brain:  $CPU_OLLAMA_URL"
    echo -e "   GPU Brain:  $GPU_OLLAMA_URL (default)"
    echo ""
}

main "$@"
