#!/usr/bin/env python3
"""
Context-aware ask: Gathers project context, then asks Ollama.
This is the "conductor" that creates a Claude-like experience by chaining
context gathering + LLM call.
"""
import sys, os, json, subprocess, urllib.request, argparse
from pathlib import Path

OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434")
MODEL = os.environ.get("OLLAMA_MODEL", "llama3-groq-tool-use:8b")

def run(cmd: str) -> str:
    """Run shell command, return output."""
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=10)
        return result.stdout.strip() or result.stderr.strip()
    except:
        return ""

def gather_git_context() -> dict:
    """Gather git repository context."""
    ctx = {}

    # Check if we're in a git repo
    if run("git rev-parse --git-dir 2>/dev/null"):
        ctx["branch"] = run("git branch --show-current")
        ctx["status"] = run("git status --short")
        ctx["recent_commits"] = run("git log --oneline -5")
        ctx["staged_diff"] = run("git diff --cached --stat")
        ctx["unstaged_diff"] = run("git diff --stat")

    return ctx

def gather_file_context(paths: list = None) -> str:
    """Read key project files for context."""
    content = []

    # Default files to check
    check_files = [
        "README.md", "README", "readme.md",
        "pyproject.toml", "package.json", "Cargo.toml", "go.mod",
        "Makefile", "justfile",
    ]

    # Add explicitly requested paths
    if paths:
        check_files = paths + check_files

    for fname in check_files:
        p = Path(fname)
        if p.exists() and p.is_file():
            try:
                text = p.read_text()[:2000]  # First 2KB
                content.append(f"### {fname}\n{text}\n")
            except:
                pass

    return "\n".join(content[:3])  # Max 3 files

def gather_directory_context() -> str:
    """Get directory structure overview."""
    try:
        # Get top-level structure
        result = subprocess.run(
            "find . -maxdepth 2 -type f -name '*.py' -o -name '*.rs' -o -name '*.js' -o -name '*.ts' -o -name '*.go' 2>/dev/null | head -20",
            shell=True, capture_output=True, text=True
        )
        files = result.stdout.strip()

        # Also get directories
        dirs = run("ls -d */ 2>/dev/null | head -10")

        return f"Directories: {dirs}\n\nCode files:\n{files}"
    except:
        return ""

def ask_ollama(question: str, context: str, model: str) -> str:
    """Send question with context to Ollama."""
    prompt = f"""You are a helpful coding assistant. Answer questions about this project using the context provided.

## Project Context
{context}

## Question
{question}

## Answer (be concise and specific)"""

    payload = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "stream": False
    }

    req = urllib.request.Request(
        f"{OLLAMA_URL}/api/chat",
        json.dumps(payload).encode(),
        {"Content-Type": "application/json"}
    )

    with urllib.request.urlopen(req, timeout=300) as r:
        resp = json.loads(r.read())
        return resp.get("message", {}).get("content", "No response")

def main():
    parser = argparse.ArgumentParser(description="Ask questions with project context")
    parser.add_argument("question", nargs="+", help="Your question")
    parser.add_argument("-m", "--model", default=MODEL, help=f"Model (default: {MODEL})")
    parser.add_argument("-f", "--files", nargs="*", help="Additional files to include")
    parser.add_argument("-c", "--context-only", action="store_true", help="Just show gathered context")
    parser.add_argument("-v", "--verbose", action="store_true", help="Show what context is gathered")
    args = parser.parse_args()

    question = " ".join(args.question)

    # Gather context from multiple sources
    print("Gathering context...", file=sys.stderr)

    git_ctx = gather_git_context()
    file_ctx = gather_file_context(args.files)
    dir_ctx = gather_directory_context()

    # Build context string
    context_parts = []

    if git_ctx:
        git_str = f"""### Git Status
Branch: {git_ctx.get('branch', 'N/A')}
Status: {git_ctx.get('status', 'clean')}
Recent commits:
{git_ctx.get('recent_commits', 'N/A')}"""
        context_parts.append(git_str)

    if dir_ctx:
        context_parts.append(f"### Project Structure\n{dir_ctx}")

    if file_ctx:
        context_parts.append(f"### Key Files\n{file_ctx}")

    full_context = "\n\n".join(context_parts) or "No project context available."

    if args.context_only:
        print(full_context)
        return

    if args.verbose:
        print("--- Context gathered ---", file=sys.stderr)
        print(full_context[:500] + "...", file=sys.stderr)
        print("------------------------", file=sys.stderr)

    print(f"Asking {args.model}...", file=sys.stderr)
    print(ask_ollama(question, full_context, args.model))

if __name__ == "__main__":
    main()
