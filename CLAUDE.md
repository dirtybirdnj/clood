# Claude Agent Guidelines

Guidelines for Claude agents working in this repository.

## Installation & System Changes

**Always ask before installing software.** Never run:
- `brew install ...`
- `apt install ...`
- `pip install ...`
- `npm install -g ...`
- `cargo install ...`
- Any package manager or installer

Instead, tell the user what needs to be installed and let them decide:

```
To run the benchmarks, Ollama needs to be installed.
Would you like me to provide installation instructions, or would you prefer to install it yourself?

Install command: brew install ollama
```

This applies to:
- Package managers (brew, apt, yum, pacman, pip, npm, cargo)
- curl/wget installers
- Any command that modifies the system outside the project directory

## Git Operations

- Commit messages should include a haiku or creative element when documenting hardware/benchmarks
- Always run `git status` before making commits
- Never force push to main

## Benchmarking

When running benchmarks on new hardware:
1. Record the hostname and hardware specs
2. Run all standard models (tinyllama, qwen2.5-coder:3b, llama3.1:8b, qwen2.5-coder:7b)
3. Add results to `ollama-tuning.md` benchmark table
4. Include a creative element (haiku, limerick, shanty) about the hardware

## Documentation Style

- Keep documentation practical and copy-paste friendly
- Include actual commands, not abstract descriptions
- Add troubleshooting sections for common issues
