#!/bin/bash
# Clood aliases - source this file: source ~/Code/clood/scripts/clood-aliases.sh

CLOOD_SCRIPTS="$HOME/Code/clood/scripts"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
OLLAMA_MODEL="${OLLAMA_MODEL:-llama3-groq-tool-use:8b}"

# Core tool aliases
alias clood-review='python3 $CLOOD_SCRIPTS/code-review.py'
alias clood-edit='python3 $CLOOD_SCRIPTS/code-review.py --edit'
alias clood-search='python3 $CLOOD_SCRIPTS/search-ask.py'
alias clood-gh='python3 $CLOOD_SCRIPTS/gh-ask.py'

# Quick mods shortcuts (single-shot transforms)
alias clood-explain='mods "explain this concisely"'
alias clood-fix='mods "fix any bugs in this code, output only the fixed code"'
alias clood-summarize='mods "summarize in 3 bullet points"'

# Git workflow aliases
alias clood-diff='git diff | mods "review these changes, list any issues"'
alias clood-staged='git diff --cached | mods "review staged changes, list any issues"'
alias clood-commit-msg='git diff --cached | mods "write a concise commit message, imperative mood, no quotes, output only the message"'
alias clood-log='git log --oneline -10 | mods "summarize recent work in one paragraph"'

# Error handling
alias clood-error='mods "explain this error and suggest a fix"'

# Chained workflows (functions for more control)

# Review a file then optionally edit
clood-review-edit() {
    local file="$1"
    if [[ -z "$file" ]]; then
        echo "Usage: clood-review-edit <file>"
        return 1
    fi
    echo "Reviewing $file..."
    python3 "$CLOOD_SCRIPTS/code-review.py" "$file"
    echo ""
    read -p "Apply interactive edits? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        python3 "$CLOOD_SCRIPTS/code-review.py" "$file" --edit
    fi
}

# Commit with AI-generated message
clood-commit() {
    if ! git diff --cached --quiet; then
        echo "Generating commit message..."
        local msg=$(git diff --cached | mods "write a concise commit message for this diff. imperative mood. no quotes. output only the message, nothing else.")
        echo "Proposed: $msg"
        read -p "Use this message? [Y/n/e(dit)] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            echo "Aborted."
        elif [[ $REPLY =~ ^[Ee]$ ]]; then
            git commit -e -m "$msg"
        else
            git commit -m "$msg"
        fi
    else
        echo "Nothing staged to commit."
    fi
}

# Search web then ask follow-up (chained)
clood-research() {
    local query="$*"
    if [[ -z "$query" ]]; then
        echo "Usage: clood-research <question>"
        return 1
    fi
    python3 "$CLOOD_SCRIPTS/search-ask.py" "$query"
}

# Context-aware ask: gathers git + file context, then asks
clood-ask() {
    local question="$*"
    if [[ -z "$question" ]]; then
        echo "Usage: clood-ask <question about current project>"
        return 1
    fi
    python3 "$CLOOD_SCRIPTS/context-ask.py" "$question"
}

# Quick file explanation
clood-explain-file() {
    local file="$1"
    if [[ -z "$file" ]]; then
        echo "Usage: clood-explain-file <file>"
        return 1
    fi
    cat "$file" | mods "explain what this code does in 2-3 sentences"
}

# Print available commands
clood-help() {
    echo "Clood Aliases - Local LLM Productivity Tools"
    echo ""
    echo "Core Tools:"
    echo "  clood-review <path>      Review code (prose output)"
    echo "  clood-edit <file>        Interactive edit mode"
    echo "  clood-search <query>     Web search + LLM answer"
    echo "  clood-gh <question>      Ask about GitHub repo"
    echo ""
    echo "Quick Transforms (pipe to these):"
    echo "  clood-explain            Explain piped content"
    echo "  clood-fix                Fix bugs in piped code"
    echo "  clood-summarize          Summarize in bullets"
    echo "  clood-error              Explain error + suggest fix"
    echo ""
    echo "Git Workflow:"
    echo "  clood-diff               Review unstaged changes"
    echo "  clood-staged             Review staged changes"
    echo "  clood-commit-msg         Generate commit message"
    echo "  clood-commit             Commit with AI message"
    echo "  clood-log                Summarize recent commits"
    echo ""
    echo "Chained Workflows:"
    echo "  clood-review-edit <file> Review then optionally edit"
    echo "  clood-research <query>   Web search + answer"
    echo "  clood-explain-file <f>   Quick file explanation"
    echo ""
    echo "Environment:"
    echo "  OLLAMA_URL=$OLLAMA_URL"
    echo "  OLLAMA_MODEL=$OLLAMA_MODEL"
}

echo "Clood aliases loaded. Run 'clood-help' for commands."
