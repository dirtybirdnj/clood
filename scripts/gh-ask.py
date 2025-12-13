#!/usr/bin/env python3
"""Query GitHub repos with gh CLI, feed to Ollama."""
import sys, json, subprocess, urllib.request, argparse

OLLAMA_URL = "http://localhost:11434"
MODEL = "llama3-groq-tool-use:8b"

def gh(cmd: str) -> str:
    """Run gh command and return output."""
    result = subprocess.run(f"gh {cmd}", shell=True, capture_output=True, text=True)
    return result.stdout or result.stderr

def gather_context(repo: str = None) -> dict:
    """Gather context from gh CLI."""
    ctx = {}

    # If repo specified, set it
    prefix = f"-R {repo} " if repo else ""

    # Recent issues
    ctx["issues"] = gh(f"{prefix}issue list --limit 10 --json number,title,state,author")

    # Recent PRs
    ctx["prs"] = gh(f"{prefix}pr list --limit 10 --json number,title,state,author,isDraft")

    # Repo info
    if repo:
        ctx["repo"] = gh(f"repo view {repo} --json name,description,stargazerCount,forkCount,primaryLanguage")
    else:
        ctx["repo"] = gh("repo view --json name,description,stargazerCount,forkCount,primaryLanguage")

    # Recent commits
    ctx["commits"] = gh(f"{prefix}api repos/:owner/:repo/commits --jq '.[0:5] | .[] | .commit.message' 2>/dev/null") or "N/A"

    return ctx

def ask(question: str, context: dict, model: str, ollama_url: str) -> str:
    """Ask Ollama with GitHub context."""
    ctx_text = f"""## Repository Info
{context.get('repo', 'N/A')}

## Recent Issues
{context.get('issues', 'None')}

## Recent PRs
{context.get('prs', 'None')}

## Recent Commits
{context.get('commits', 'N/A')}"""

    prompt = f"""You are a GitHub assistant. Answer questions about this repository.

{ctx_text}

## Question
{question}

## Answer"""

    payload = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "stream": False
    }

    req = urllib.request.Request(
        f"{ollama_url}/api/chat",
        json.dumps(payload).encode(),
        {"Content-Type": "application/json"}
    )

    with urllib.request.urlopen(req, timeout=300) as r:
        return json.loads(r.read()).get("message", {}).get("content", "No response")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Ask questions about GitHub repos")
    parser.add_argument("question", nargs="+", help="Your question")
    parser.add_argument("-r", "--repo", help="Repository (owner/repo)")
    parser.add_argument("-m", "--model", default=MODEL)
    parser.add_argument("-o", "--ollama", default=OLLAMA_URL)
    parser.add_argument("-c", "--context-only", action="store_true", help="Just show context")
    args = parser.parse_args()

    print("📦 Gathering GitHub context...", file=sys.stderr)
    ctx = gather_context(args.repo)

    if args.context_only:
        print(json.dumps(ctx, indent=2))
    else:
        print(f"🤖 Asking {args.model}...", file=sys.stderr)
        print(ask(" ".join(args.question), ctx, args.model, args.ollama))
