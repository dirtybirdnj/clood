#!/usr/bin/env python3
"""Review code at a path using Ollama directly."""
import sys, os, json, urllib.request, argparse
from pathlib import Path

OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434")
MODEL = os.environ.get("OLLAMA_MODEL", "llama3-groq-tool-use:8b")

EXTENSIONS = {'.py', '.js', '.ts', '.tsx', '.go', '.rs', '.rb', '.sh', '.md', '.json', '.yaml', '.yml'}
SKIP_DIRS = {'.git', 'node_modules', '__pycache__', 'venv', '.venv', 'dist', 'build'}

def read_files(path: str, max_size: int = 50000) -> str:
    """Read all code files at path into a single string."""
    p = Path(path)
    content = []
    total = 0

    if p.is_file():
        files = [p]
    else:
        files = sorted(p.rglob('*'))

    for f in files:
        if any(skip in f.parts for skip in SKIP_DIRS):
            continue
        if f.is_file() and f.suffix in EXTENSIONS:
            try:
                text = f.read_text()
                if total + len(text) > max_size:
                    content.append(f"\n### {f} (truncated - size limit)\n")
                    break
                content.append(f"\n### {f}\n```{f.suffix[1:]}\n{text}\n```\n")
                total += len(text)
            except: pass

    return "".join(content) or "No code files found."

def review(path: str, model: str = MODEL) -> str:
    """Send code to Ollama for review."""
    code = read_files(path)

    prompt = f"""Review the following code. Be specific and actionable.

## Your Task
1. Give a brief summary of what this code does (2-3 sentences)
2. List any bugs, issues, or red flags you see
3. Provide exactly 3 improvement ideas with code samples

## Code to Review
{code}

## Your Review"""

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

    print(f"Reviewing {path} with {model}...", file=sys.stderr)

    with urllib.request.urlopen(req, timeout=300) as r:
        resp = json.loads(r.read())
        return resp.get("message", {}).get("content", "No response")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Review code with Ollama")
    parser.add_argument("path", help="Path to file or directory to review")
    parser.add_argument("-m", "--model", default=MODEL, help=f"Model to use (default: {MODEL})")
    parser.add_argument("-u", "--url", default=OLLAMA_URL, help=f"Ollama URL (default: {OLLAMA_URL})")
    args = parser.parse_args()

    OLLAMA_URL = args.url
    print(review(args.path, args.model))
