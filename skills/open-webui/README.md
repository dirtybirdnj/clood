# Open WebUI Tools & Functions

Python tools and functions for Open WebUI integration with clood.

## clood-tools.py - Codebase Exploration

MCP-like tools that let LLMs explore your codebase:

| Tool | Description |
|------|-------------|
| `grep(pattern)` | Search codebase with regex |
| `tree(path)` | Show directory structure |
| `symbols(path)` | Extract functions, types, classes |
| `imports(file)` | Analyze dependencies |
| `context(path)` | Generate LLM-optimized project summary |
| `ask(question, model)` | Query another model (inception) |
| `hosts()` | List Ollama hosts |
| `models()` | List available models |
| `git_diff(ref)` | Show git changes |
| `git_log(count)` | Show recent commits |

### Installation

1. Open http://localhost:3000
2. Go to **Workspace** > **Tools**
3. Click **+** (Add Tool)
4. Paste contents of `clood-tools.py`
5. Click **Save**

### Configuration

After importing, configure the tool valves (settings):

- **CLOOD_PATH**: Path to clood binary (default: `clood`)
- **WORKING_DIR**: Default working directory for commands

### Usage

In chat with a tool-capable model:
```
What functions are in the router package?
```

The model calls `symbols("internal/router")` to answer.

Or explicitly request a tool:
```
Use the grep tool to find all usages of "hostMgr"
```

### Notes

- Tools execute via subprocess, calling the clood CLI
- Results are returned to the model as context
- Works best with tool-capable models (llama3-groq-tool-use, qwen2.5)

---

## Adding Custom Tools

1. Save the Python file here with a descriptive name
2. In Open WebUI: Workspace > Tools > Create
3. Paste the code and configure

## Exporting Existing Tools

```bash
# Copy database from container
docker cp open-webui:/app/backend/data/webui.db /tmp/webui.db

# Run export script
python3 ../scripts/export-openwebui-tools.py /tmp/webui.db ./
```

## Tool Template

```python
"""
title: My Tool Name
author: you
version: 0.1.0
"""

from typing import Callable, Any

class Tools:
    def __init__(self):
        self.valves = self.Valves()

    class Valves:
        MY_SETTING: str = "default_value"

    def my_function(
        self,
        param: str,
        __event_emitter__: Callable[[dict], Any] = None
    ) -> str:
        """
        Description of what this tool does.

        :param param: Description of parameter
        :return: What it returns
        """
        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Working...", "done": False}})

        result = f"Result: {param}"

        if __event_emitter__:
            __event_emitter__({"type": "status", "data": {"description": "Done", "done": True}})

        return result
```
