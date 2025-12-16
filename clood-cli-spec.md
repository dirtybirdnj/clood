Here is the CLI spec for the unified `clood` command:
```bash
clood-cli-spec.md
# clood

## Command: clood
### Synopsis:
Unified CLI command that combines review, search, gh, and context-ask functionality.

### Options:

* `-r`: Run a code review.
* `-s`: Perform a web search.
* `-g`: Ask GitHub for information.
* `-c`: Ask the context about something.

### Examples:

```bash
# Code Review
clood -r

# Web Search
clood -s "What is AI?"

# GitHub Information
clood -g "github.com"

# Context Ask
clood -c "What's the weather like today?"
```

This unified command simplifies interactions with your LLM toolkit, providing a single entry point for various tasks.
