# Local LLM Prompt Templates

Prompt templates designed for small local models (7B and under).

## Files

| Template | Use Case |
|----------|----------|
| `add-docstring.md` | Add Google-style docstrings to functions |
| `implement-todos.md` | Fill in TODO comments in code |
| `security-checklist.md` | Yes/No security review questions |
| `convert-callback-async.md` | Transform callbacks to async/await |
| `write-regex.md` | Generate regex patterns from descriptions |
| `type-annotations.md` | Add Python type hints to code |

## How to Use

1. Open template in editor
2. Replace `{PLACEHOLDER}` with your code
3. Paste entire prompt into Crush
4. Copy output

## Tips

- Use qwen2.5-coder for code tasks
- Use llama3.1 for text/checklist tasks
- If output includes explanations, add "Output ONLY the code" again
- Run multiple times and pick best result
