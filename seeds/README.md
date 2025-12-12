# Seeds Directory

Claude-generated seed artifacts for local LLM execution.

## Subdirectories

### `skeletons/`
Code skeletons with TODO comments. Local LLM fills in implementations.

### `tests/`
Test-first specs. Claude writes tests, local LLM writes code to pass them.

### `checklists/`
Yes/No review checklists. Local LLM answers each question about code.

### `transforms/`
Input→Output transformation specs with rules and examples.

## Usage

1. Claude creates a seed artifact here
2. Copy/paste into Crush with target code
3. Local LLM produces output
4. Review and iterate

## Creating New Seeds

Use Claude to generate seeds. Be specific:
- "Create a skeleton for a REST API that handles user authentication"
- "Write tests for a function that parses CSV with headers"
- "Create a checklist for reviewing React components"
