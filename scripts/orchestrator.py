#!/usr/bin/env python3
"""
Orchestrator Agent - A local AI agent that can:
- Read/write files
- Run git commands
- Delegate coding tasks to more powerful models

This runs on ubuntu25 and coordinates the server garden.
"""

import json
import subprocess
import requests
import sys
from pathlib import Path

# Configuration
ORCHESTRATOR_URL = "http://localhost:11434"  # Ollama on ubuntu25

# Conductor model rankings (from catfight testing):
#
# For AGENTIC LOOPS (multi-turn tool execution):
#   1. llama3-groq-tool-use:8b - Actually invokes tools, fast (~8s) [BEST]
#   2. llama3.1:8b - Sometimes works, medium speed
#
# For SINGLE-TURN (one-shot):
#   1. mistral:7b - Best accuracy but DESCRIBES tools instead of calling them
#
# AVOID for orchestration:
#   - "coder" models (qwen2.5-coder, codestral) - bad at tool format
#
ORCHESTRATOR_MODEL = "llama3-groq-tool-use:8b"  # Best for agentic loops

CODER_HOSTS = {
    "mac-mini": "http://192.168.4.41:11434",
    "mac-laptop": "http://192.168.4.47:11434",
    "ubuntu25-fast": "http://localhost:8080",  # llama.cpp
}

CODER_MODEL = "qwen2.5-coder:7b"  # Default coding model
WORKSPACE = Path("/data/repos/workspace")  # Where code lives on ubuntu25

# Tool definitions (OpenAI function calling format)
TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "read_file",
            "description": "Read the contents of a file",
            "parameters": {
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Path to the file (relative to workspace)"}
                },
                "required": ["path"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "write_file",
            "description": "Write content to a file",
            "parameters": {
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Path to the file (relative to workspace)"},
                    "content": {"type": "string", "description": "Content to write"}
                },
                "required": ["path", "content"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "list_directory",
            "description": "List files in a directory",
            "parameters": {
                "type": "object",
                "properties": {
                    "path": {"type": "string", "description": "Directory path (relative to workspace)"}
                },
                "required": ["path"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "git_status",
            "description": "Get git status of the workspace",
            "parameters": {"type": "object", "properties": {}}
        }
    },
    {
        "type": "function",
        "function": {
            "name": "git_commit",
            "description": "Stage all changes and commit with a message",
            "parameters": {
                "type": "object",
                "properties": {
                    "message": {"type": "string", "description": "Commit message"}
                },
                "required": ["message"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "git_push",
            "description": "Push commits to remote",
            "parameters": {"type": "object", "properties": {}}
        }
    },
    {
        "type": "function",
        "function": {
            "name": "delegate_coding",
            "description": "Send a coding task to a more powerful model. If output_file is specified, automatically saves the code to that file.",
            "parameters": {
                "type": "object",
                "properties": {
                    "prompt": {"type": "string", "description": "The coding task to perform"},
                    "output_file": {"type": "string", "description": "Filename to save the result to (e.g. 'todo.html')"}
                },
                "required": ["prompt"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "task_complete",
            "description": "Call this when the task is fully complete",
            "parameters": {
                "type": "object",
                "properties": {
                    "summary": {"type": "string", "description": "Summary of what was accomplished"}
                },
                "required": ["summary"]
            }
        }
    }
]


def execute_tool(name: str, args: dict) -> str:
    """Execute a tool and return the result."""

    if name == "read_file":
        path = WORKSPACE / args["path"]
        if not path.exists():
            return f"Error: File not found: {args['path']}"
        return path.read_text()

    elif name == "write_file":
        # Handle paths - strip leading slash and any absolute path attempts
        file_path = args["path"].lstrip("/").split("/")[-1]  # Just get filename
        path = WORKSPACE / file_path
        path.parent.mkdir(parents=True, exist_ok=True)
        # Strip markdown code fences if present
        content = args["content"]
        if content.startswith("```"):
            lines = content.split("\n")
            content = "\n".join(lines[1:-1] if lines[-1].startswith("```") else lines[1:])
        path.write_text(content)
        return f"Successfully wrote {len(content)} bytes to {file_path}"

    elif name == "list_directory":
        path = WORKSPACE / args.get("path", ".")
        if not path.exists():
            return f"Error: Directory not found: {args['path']}"
        files = list(path.iterdir())
        return "\n".join(f.name for f in files)

    elif name == "git_status":
        result = subprocess.run(
            ["git", "status", "--short"],
            cwd=WORKSPACE,
            capture_output=True,
            text=True
        )
        return result.stdout or "Nothing to commit, working tree clean"

    elif name == "git_commit":
        # Stage all changes
        subprocess.run(["git", "add", "-A"], cwd=WORKSPACE)
        # Commit
        result = subprocess.run(
            ["git", "commit", "-m", args["message"]],
            cwd=WORKSPACE,
            capture_output=True,
            text=True
        )
        return result.stdout + result.stderr

    elif name == "git_push":
        result = subprocess.run(
            ["git", "push"],
            cwd=WORKSPACE,
            capture_output=True,
            text=True
        )
        return result.stdout + result.stderr or "Pushed successfully"

    elif name == "delegate_coding":
        prompt = args["prompt"]
        output_file = args.get("output_file")

        # Always use ubuntu25-fast (llama.cpp is 4x faster for 7B)
        url = f"{CODER_HOSTS['ubuntu25-fast']}/v1/chat/completions"
        payload = {
            "messages": [
                {"role": "system", "content": "You are an expert programmer. Output only clean, working code with no explanations."},
                {"role": "user", "content": prompt}
            ],
            "max_tokens": 2000
        }
        try:
            resp = requests.post(url, json=payload, timeout=120)
            result = resp.json()
            code = result["choices"][0]["message"]["content"]

            # Strip markdown code fences if present
            if code.startswith("```"):
                lines = code.split("\n")
                code = "\n".join(lines[1:-1] if lines[-1].startswith("```") else lines[1:])

            # Auto-save if output_file specified
            if output_file:
                path = WORKSPACE / output_file.lstrip("/").split("/")[-1]
                path.write_text(code)
                return f"Generated and saved {len(code)} bytes to {output_file}"

            return code
        except Exception as e:
            return f"Error calling llama.cpp: {e}"

    elif name == "task_complete":
        return f"TASK_COMPLETE: {args['summary']}"

    return f"Unknown tool: {name}"


def call_orchestrator(messages: list, model: str = None) -> dict:
    """Call the orchestrator model with tools."""
    model = model or ORCHESTRATOR_MODEL

    # Ollama's tool calling format
    payload = {
        "model": model,
        "messages": messages,
        "tools": TOOLS,
        "stream": False
    }

    resp = requests.post(
        f"{ORCHESTRATOR_URL}/api/chat",
        json=payload,
        timeout=60
    )

    return resp.json()


def run_agent(task: str, max_iterations: int = 10, conductor: str = None):
    """Run the agent loop."""
    model = conductor or ORCHESTRATOR_MODEL

    print(f"\n{'='*60}")
    print(f"TASK: {task}")
    print(f"CONDUCTOR: {model}")
    print(f"{'='*60}\n")

    messages = [
        {
            "role": "system",
            "content": """You are an AI coding agent. You MUST use tools to complete tasks.

IMPORTANT: When creating files, ALWAYS use delegate_coding with output_file parameter.
This automatically saves the code - no separate write_file needed.

Example - to create todo.html:
  delegate_coding(prompt="Create a todo list HTML", output_file="todo.html")
  task_complete(summary="Created todo.html")

Available tools:
- delegate_coding(prompt, output_file): Generate AND save code in one step
- read_file(path): Read a file
- git_commit(message): Commit changes
- task_complete(summary): Signal done

DO NOT call write_file after delegate_coding - use output_file instead."""
        },
        {"role": "user", "content": task}
    ]

    for i in range(max_iterations):
        print(f"--- Iteration {i+1} ---")

        response = call_orchestrator(messages, model)
        message = response.get("message", {})

        # Check for tool calls
        tool_calls = message.get("tool_calls", [])

        if not tool_calls:
            # No tool calls - model is just responding
            content = message.get("content", "")
            print(f"Agent: {content}")

            # Only stop if explicitly complete or empty after many iterations
            if "TASK_COMPLETE" in content:
                print("\n✓ Task finished")
                break

            if not content and i > 5:
                print("\n⚠ Agent stopped responding")
                break

            messages.append(message)

            # If there's content but no tool call, prompt to continue
            if content and "?" not in content:
                messages.append({"role": "user", "content": "Continue with the task. Use tools to complete it."})
            continue

        # Execute tool calls
        messages.append(message)

        for tool_call in tool_calls:
            func = tool_call.get("function", {})
            name = func.get("name", "")
            args = func.get("arguments", {})

            # Parse args if string
            if isinstance(args, str):
                try:
                    args = json.loads(args)
                except:
                    args = {}

            print(f"  Tool: {name}({json.dumps(args)[:100]}...)")

            result = execute_tool(name, args)

            if "TASK_COMPLETE" in result:
                print(f"\n✓ {result}")
                return

            print(f"  Result: {result[:200]}{'...' if len(result) > 200 else ''}")

            # Add tool result to messages
            messages.append({
                "role": "tool",
                "content": result
            })

    print("\n⚠ Max iterations reached")


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(
        description="Orchestrator Agent - Local AI coding assistant",
        epilog="""
Conductor models (ranked by tool-calling ability):
  mistral:7b              Best accuracy, slower (~28s)
  llama3-groq-tool-use:8b Good balance, fast (~8s) [default]
  llama3.1:8b             Decent, medium speed

Examples:
  python orchestrator.py 'Create a todo list HTML file'
  python orchestrator.py --conductor mistral:7b 'Create hello.html'
"""
    )
    parser.add_argument("task", nargs="*", help="The task to perform")
    parser.add_argument("--conductor", "-c", default=ORCHESTRATOR_MODEL,
                       help="Conductor model to use (default: llama3-groq-tool-use:8b)")
    parser.add_argument("--max-iterations", "-m", type=int, default=10,
                       help="Max iterations (default: 10)")

    args = parser.parse_args()

    if not args.task:
        parser.print_help()
        sys.exit(1)

    task = " ".join(args.task)
    run_agent(task, max_iterations=args.max_iterations, conductor=args.conductor)
