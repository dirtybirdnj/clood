#!/usr/bin/env python3
"""
Conductor Catfight - Test different models as orchestration "conductors"

Tests each model's ability to:
1. Understand the task
2. Use tools correctly (especially output_file parameter)
3. Complete tasks without excessive iterations
4. Avoid truncation and lazy behavior

Usage:
    python3 conductor-catfight.py
"""

import json
import subprocess
import time
from pathlib import Path

# Configuration
OLLAMA_URLS = {
    "ubuntu25": "http://192.168.4.64:11434",
    "mac-mini": "http://192.168.4.41:11434",
    "mac-laptop": "http://192.168.4.47:11434",
}

# Conductor candidates to test (model, host)
# Based on Docker's tool-calling benchmark and Ollama tool support
CONDUCTORS = [
    # Tier 1: Proven tool-calling performers
    ("qwen3:8b", "ubuntu25"),           # F1=0.933 in Docker benchmark
    ("hermes3:8b", "ubuntu25"),         # NousResearch, trained for tool-use
    ("command-r:7b", "ubuntu25"),       # Cohere's agentic model
    ("llama3-groq-tool-use:8b", "ubuntu25"),  # Current default

    # Tier 2: General purpose with tool support
    ("llama3.1:8b", "ubuntu25"),        # F1=0.835 in Docker benchmark
    ("mistral:7b", "ubuntu25"),         # Good single-turn, bad multi-turn

    # Tier 3: Coder models (expected to fail at orchestration)
    ("qwen2.5-coder:7b", "ubuntu25"),   # For comparison - should be bad

    # Bigger models on laptop (if available)
    ("qwen3:14b", "mac-laptop"),        # F1=0.971 - near GPT-4!
    ("llama3-groq-tool-use:8b", "mac-laptop"),
    ("qwen2.5-coder:32b", "mac-laptop"),
]

# Simple tool definition for testing
TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "delegate_coding",
            "description": "Generate code and save to file",
            "parameters": {
                "type": "object",
                "properties": {
                    "prompt": {"type": "string", "description": "The coding task"},
                    "output_file": {"type": "string", "description": "Filename to save"}
                },
                "required": ["prompt", "output_file"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "task_complete",
            "description": "Signal task is done",
            "parameters": {
                "type": "object",
                "properties": {
                    "summary": {"type": "string", "description": "What was accomplished"}
                },
                "required": ["summary"]
            }
        }
    }
]

SYSTEM_PROMPT = """You are an AI coding agent. Use tools to complete tasks.

To create a file:
  delegate_coding(prompt="description", output_file="filename.ext")

When done:
  task_complete(summary="what you did")

ALWAYS use output_file parameter with delegate_coding."""

TEST_TASK = "Create a hello.html file that displays 'Hello World' in a styled heading"


def test_conductor(model: str, host: str) -> dict:
    """Test a single conductor model."""
    url = f"{OLLAMA_URLS[host]}/api/chat"

    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": TEST_TASK}
    ]

    payload = json.dumps({
        "model": model,
        "messages": messages,
        "tools": TOOLS,
        "stream": False
    })

    start = time.time()

    try:
        proc = subprocess.run(
            ["curl", "-s", "-X", "POST", url,
             "-H", "Content-Type: application/json",
             "-d", payload],
            capture_output=True,
            text=True,
            timeout=60
        )

        elapsed = time.time() - start
        result = json.loads(proc.stdout)
        message = result.get("message", {})
        tool_calls = message.get("tool_calls", [])
        content = message.get("content", "")

        # Analyze the response
        used_delegate = False
        used_output_file = False
        used_task_complete = False

        for tc in tool_calls:
            func = tc.get("function", {})
            name = func.get("name", "")
            args = func.get("arguments", {})
            if isinstance(args, str):
                try:
                    args = json.loads(args)
                except:
                    args = {}

            if name == "delegate_coding":
                used_delegate = True
                if args.get("output_file"):
                    used_output_file = True
            elif name == "task_complete":
                used_task_complete = True

        # Score the conductor
        score = 0
        if used_delegate:
            score += 30
        if used_output_file:
            score += 40  # Most important!
        if used_task_complete:
            score += 20
        if elapsed < 10:
            score += 10  # Speed bonus

        return {
            "model": model,
            "host": host,
            "success": used_delegate and used_output_file,
            "used_delegate": used_delegate,
            "used_output_file": used_output_file,
            "used_task_complete": used_task_complete,
            "tool_calls": len(tool_calls),
            "elapsed": round(elapsed, 2),
            "score": score,
            "error": None
        }

    except Exception as e:
        return {
            "model": model,
            "host": host,
            "success": False,
            "error": str(e),
            "score": 0
        }


def main():
    print("=" * 70)
    print("  CONDUCTOR CATFIGHT - Testing Orchestrator Models")
    print("=" * 70)
    print(f"\nTask: {TEST_TASK}")
    print(f"Testing {len(CONDUCTORS)} conductor candidates...\n")

    results = []

    for model, host in CONDUCTORS:
        print(f"Testing {model} on {host}...", end=" ", flush=True)
        result = test_conductor(model, host)
        results.append(result)

        if result.get("error"):
            print(f"ERROR: {result['error'][:50]}")
        else:
            status = "✓" if result["success"] else "✗"
            print(f"{status} score={result['score']} time={result['elapsed']}s")

    # Sort by score
    results.sort(key=lambda x: x.get("score", 0), reverse=True)

    print("\n" + "=" * 70)
    print("  RESULTS (sorted by score)")
    print("=" * 70)
    print(f"{'Model':<30} {'Host':<12} {'Score':<8} {'output_file':<12} {'Time':<8}")
    print("-" * 70)

    for r in results:
        if r.get("error"):
            print(f"{r['model']:<30} {r['host']:<12} {'ERR':<8} {'-':<12} {'-':<8}")
        else:
            of = "✓" if r["used_output_file"] else "✗"
            print(f"{r['model']:<30} {r['host']:<12} {r['score']:<8} {of:<12} {r['elapsed']:<8}")

    # Winner announcement
    winner = results[0] if results else None
    if winner and not winner.get("error"):
        print(f"\n🏆 WINNER: {winner['model']} on {winner['host']} (score: {winner['score']})")

    print()


if __name__ == "__main__":
    main()
