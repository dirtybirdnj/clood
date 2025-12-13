#!/bin/bash

# ==============================================================================
# CLOOD FLOW - Split Brain Query Router
# ==============================================================================
#
# The Flow
# Query enters the stream
# CPU whispers: simple, complex?
# GPU awakens
# ==============================================================================

# Source common configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/clood-common.sh"

# ==============================================================================
# USAGE
# ==============================================================================

usage() {
    echo "Usage: clood-flow.sh [OPTIONS] <query>"
    echo ""
    echo "Routes queries to appropriate tier based on complexity."
    echo ""
    echo "Options:"
    echo "  -t, --tier cpu|gpu    Force specific tier"
    echo "  -m, --model MODEL     Override model selection"
    echo "  -v, --verbose         Show routing decision"
    echo "  -s, --stream          Stream response (default: buffered)"
    echo "  -h, --help            Show this help"
    echo ""
    echo "Examples:"
    echo "  clood-flow.sh 'What is 2+2?'"
    echo "  clood-flow.sh -t gpu 'Explain quantum computing'"
    echo "  clood-flow.sh -v 'Refactor this to use async/await'"
    echo ""
}

# ==============================================================================
# MAIN LOGIC
# ==============================================================================

main() {
    local force_tier=""
    local force_model=""
    local verbose=false
    local stream=false
    local query=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -t|--tier)
                force_tier="$2"
                shift 2
                ;;
            -m|--model)
                force_model="$2"
                shift 2
                ;;
            -v|--verbose)
                verbose=true
                shift
                ;;
            -s|--stream)
                stream=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            -*)
                log_tier error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                query="$*"
                break
                ;;
        esac
    done

    # Validate query
    if [ -z "$query" ]; then
        log_tier error "No query provided"
        usage
        exit 1
    fi

    # Determine tier
    local tier
    if [ -n "$force_tier" ]; then
        tier="$force_tier"
    else
        tier=$(select_tier "$query")
    fi

    # Select model and host based on tier
    local host model
    case "$tier" in
        cpu|1)
            host="$CPU_OLLAMA_HOST"
            model="${force_model:-$MODEL_ROUTER}"
            ;;
        gpu|2)
            host="$GPU_OLLAMA_HOST"
            model="${force_model:-$MODEL_SMART}"
            ;;
        *)
            log_tier error "Invalid tier: $tier"
            exit 1
            ;;
    esac

    # Verbose output
    if [ "$verbose" = true ]; then
        local complexity=$(classify_query "$query")
        log_tier info "Query: ${query:0:50}..."
        log_tier info "Complexity: $complexity"
        log_tier "$tier" "Routing to $tier tier ($host)"
        log_tier info "Model: $model"
        echo ""
    fi

    # Check if service is available
    if ! check_service "http://$host/api/tags" 3; then
        log_tier error "Tier $tier ($host) is not responding"

        # Fallback logic
        if [ "$tier" = "cpu" ]; then
            log_tier warn "Falling back to GPU tier"
            host="$GPU_OLLAMA_HOST"
            model="$MODEL_SMART"
            if ! check_service "http://$host/api/tags" 3; then
                log_tier error "GPU tier also unavailable. Is Ollama running?"
                exit 1
            fi
        else
            log_tier error "Run './scripts/start-cpu-services.sh' to start services"
            exit 1
        fi
    fi

    # Execute query
    if [ "$stream" = true ]; then
        # Streaming mode
        curl -s "http://$host/api/generate" \
            -d "{\"model\": \"$model\", \"prompt\": \"$query\", \"stream\": true}" \
            | while read -r line; do
                echo "$line" | jq -r '.response // empty' 2>/dev/null | tr -d '\n'
            done
        echo ""
    else
        # Buffered mode
        local response
        response=$(curl -s "http://$host/api/generate" \
            -d "{\"model\": \"$model\", \"prompt\": \"$query\", \"stream\": false}")

        # Check for errors
        local error
        error=$(echo "$response" | jq -r '.error // empty')
        if [ -n "$error" ]; then
            log_tier error "API Error: $error"
            exit 1
        fi

        # Output response
        echo "$response" | jq -r '.response // "No response"'
    fi
}

# ==============================================================================
# ENTRY POINT
# ==============================================================================

# Allow sourcing without execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
