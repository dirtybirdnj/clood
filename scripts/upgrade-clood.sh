#!/bin/bash

# Configuration
ARCHITECT_MODEL="deepseek-r1:8b"
BUILDER_MODEL="qwen2.5-coder:14b"
OUTPUT_DIR="./scripts"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${BLUE}⚡ Starting CLOOD UPGRADE PROTOCOL ⚡${NC}"
echo -e "Architect: $ARCHITECT_MODEL | Builder: $BUILDER_MODEL"

# Ensure output dir exists
mkdir -p $OUTPUT_DIR

# --- DEFINE THE MISSION ---
# We inject the prompt we designed earlier as the "Mission Context"
MISSION_CONTEXT="
You are a Senior DevOps Engineer. We are refactoring the 'clood' repository.
The Goal: Create a cohesive local LLM platform with a 'Tiered Intelligence' router.

We need to generate 4 specific files:
1. scripts/clood-common.sh: A shared library defining OLLAMA_HOST, MODEL_FAST, MODEL_SMART.
2. scripts/verify-setup.sh: A health check script that sources clood-common.sh.
3. scripts/clood-ask: The main BASH router script that uses 'mods' for simple queries and 'crush' for complex ones.
4. scripts/context-ask.py: A Python utility to fetch file context safely using XML tags.
"

# --- FUNCTION: THE GENERATOR LOOP ---
generate_file() {
    local filename=$1
    local description=$2

    echo -e "\n${PURPLE}==============================================${NC}"
    echo -e "${PURPLE}  PHASE 1: Architecting $filename...${NC}"
    echo -e "${PURPLE}==============================================${NC}"

    # 1. Architect Step: Create the Blueprint
    # We pipe the mission + specific file requirement to DeepSeek
    PLAN=$(echo "$MISSION_CONTEXT" | mods -m $ARCHITECT_MODEL -f \
    "Focus ONLY on '$filename'. $description.
    Write a detailed technical plan and pseudocode for this file.
    Do not write the final code yet, just the logic and requirements.")

    echo -e "${GREEN}✔ Plan generated.${NC}"

    echo -e "\n${BLUE}==============================================${NC}"
    echo -e "${BLUE}  PHASE 2: Building $filename...${NC}"
    echo -e "${BLUE}==============================================${NC}"

    # 2. Builder Step: Write the Code
    # We pipe the Plan to Qwen-Coder
    echo "$PLAN" | mods -m $BUILDER_MODEL -f \
    "You are an expert developer. Read the Architect's plan below.
    Write the final, working code for '$filename'.
    Output ONLY the code block. Do not use markdown backticks or explanations.
    Ensure it is valid and ready to save." > "$OUTPUT_DIR/$filename"

    # 3. Validation / Cleanup
    # Remove markdown formatting if the model added it despite instructions
    sed -i '' 's/^```.*//g' "$OUTPUT_DIR/$filename"
    sed -i '' 's/^```//g' "$OUTPUT_DIR/$filename"
    
    # Make executable if it's a script
    if [[ "$filename" == *.sh ]] || [[ "$filename" == "clood-ask" ]]; then
        chmod +x "$OUTPUT_DIR/$filename"
    fi

    echo -e "${GREEN}✔ Saved to $OUTPUT_DIR/$filename${NC}"
}

# --- EXECUTE THE UPGRADE ---

generate_file "clood-common.sh" \
"Define standard env vars (OLLAMA_HOST, MODEL_FAST=tinyllama, MODEL_SMART=llama3-groq-tool-use) and a 'log_tier' helper function for colored output."

generate_file "verify-setup.sh" \
"Source clood-common.sh. Check if 'mods' and 'crush' are in PATH. Verify config files exist."

generate_file "clood-ask" \
"Source clood-common.sh. Use 'mods' to classify input as SIMPLE or COMPLEX. If SIMPLE, pipe to mods. If COMPLEX, launch crush. Check for project context files."

generate_file "context-ask.py" \
"Refactor to use XML tags <file_context> around content to prevent prompt injection. Ensure it reads variables from clood-common.sh if possible or defaults safely."

echo -e "\n${GREEN}✨ Upgrade Complete! Check the scripts/ directory.${NC}"