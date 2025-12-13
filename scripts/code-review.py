#!/usr/bin/env python3
"""Review code at a path using Ollama directly."""
import sys, os, json, urllib.request, argparse
from pathlib import Path

OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434")
MODEL = os.environ.get("OLLAMA_MODEL", "llama3-groq-tool-use:8b")

EXTENSIONS = {'.py', '.js', '.ts', '.tsx', '.go', '.rs', '.rb', '.sh', '.md', '.json', '.yaml', '.yml', '.toml', '.html', '.css', '.c', '.h', '.cpp', '.hpp', '.java', '.swift', '.kt'}
SKIP_DIRS = {'.git', 'node_modules', '__pycache__', 'venv', '.venv', 'dist', 'build', 'target', '.claude'}

def read_files(path: str, max_size: int = 50000) -> str:
    """Read all code files at path into a single string."""
    p = Path(path)
    content = []
    total = 0

    # Prioritize code over docs
    CODE_FIRST = {'.rs', '.py', '.js', '.ts', '.go', '.c', '.cpp', '.java', '.rb'}

    if p.is_file():
        files = [p]
    else:
        all_files = list(p.rglob('*'))
        code_files = [f for f in all_files if f.suffix in CODE_FIRST]
        other_files = [f for f in all_files if f.suffix not in CODE_FIRST]
        files = sorted(code_files) + sorted(other_files)

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

REVIEW_PROMPT = """Review the following code. Be specific and actionable.

## Your Task
1. Give a brief summary of what this code does (2-3 sentences)
2. List any bugs, issues, or red flags you see
3. Provide exactly 3 improvement ideas with code samples

## Code to Review
{code}

## Your Review"""

PATCH_PROMPT = """You are a code improvement tool. Output ONLY a unified diff patch.

## Rules
- Output valid unified diff format that can be applied with `patch -p1`
- Include file paths in the diff headers
- Make exactly 3 improvements (better error handling, cleaner code, or bug fixes)
- NO explanations, NO markdown, ONLY the diff

## Code to Improve
{code}

## Unified Diff Output"""

EDIT_PROMPT = """You are a code editor. Suggest exactly 3 improvements as search/replace blocks.

## Format (use EXACTLY this format for each change):
### Change N: Brief description

<<<<<<< SEARCH
exact code to find (copy verbatim from the file)
=======
replacement code
>>>>>>> REPLACE

## Rules
- Copy the SEARCH section EXACTLY from the original code (same whitespace, same lines)
- Each block should be a focused, minimal change
- Give 3 changes maximum
- Brief 1-line description before each block

## Code to Improve
{code}

## Your Changes"""

def review(path: str, model: str = MODEL, mode: str = "review") -> str:
    """Send code to Ollama for review."""
    code = read_files(path)
    prompts = {"review": REVIEW_PROMPT, "patch": PATCH_PROMPT, "edit": EDIT_PROMPT}
    prompt = prompts.get(mode, REVIEW_PROMPT).format(code=code)

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

def apply_edits(path: str, response: str):
    """Parse SEARCH/REPLACE blocks and apply interactively."""
    import re
    blocks = re.findall(
        r'### Change \d+:([^\n]*)\n+<<<<<<< SEARCH\n(.*?)\n=======\n(.*?)\n>>>>>>> REPLACE',
        response, re.DOTALL
    )
    if not blocks:
        print("No edit blocks found in response.")
        return

    filepath = Path(path)
    if not filepath.is_file():
        print(f"Can only apply edits to a single file, got: {path}")
        return

    content = filepath.read_text()

    for i, (desc, search, replace) in enumerate(blocks, 1):
        print(f"\n{'='*60}")
        print(f"Change {i}:{desc.strip()}")
        print(f"{'='*60}")
        print(f"\033[91m- {search.strip()}\033[0m")
        print(f"\033[92m+ {replace.strip()}\033[0m")
        print()

        choice = input("Apply? [y]es [n]o [q]uit: ").lower().strip()
        if choice == 'q':
            break
        if choice == 'y':
            if search.strip() in content:
                content = content.replace(search.strip(), replace.strip(), 1)
                print("✓ Applied")
            else:
                print("✗ Could not find exact match in file")

    if input("\nSave changes? [y/n]: ").lower().strip() == 'y':
        filepath.write_text(content)
        print(f"Saved {filepath}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Review code with Ollama")
    parser.add_argument("path", help="Path to file or directory to review")
    parser.add_argument("-m", "--model", default=MODEL, help=f"Model to use (default: {MODEL})")
    parser.add_argument("-u", "--url", default=OLLAMA_URL, help=f"Ollama URL (default: {OLLAMA_URL})")
    parser.add_argument("-p", "--patch", action="store_true", help="Output unified diff")
    parser.add_argument("-e", "--edit", action="store_true", help="Interactive edit mode (Claude Code style)")
    args = parser.parse_args()

    OLLAMA_URL = args.url
    mode = "edit" if args.edit else ("patch" if args.patch else "review")
    result = review(args.path, args.model, mode=mode)

    if args.edit:
        print(result)
        apply_edits(args.path, result)
    else:
        print(result)
