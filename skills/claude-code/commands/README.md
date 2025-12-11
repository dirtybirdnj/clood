# Claude Code Slash Commands

Custom slash commands for Claude Code.

## Installation

Symlink to your Claude commands directory:

```bash
ln -s /path/to/clood/skills/claude-code/commands/*.md ~/.claude/commands/
```

Or copy individual commands you want.

## Creating Commands

Commands are markdown files. The filename (without .md) becomes the command name.

Example `review.md`:
```markdown
Review the code in $ARGUMENTS for:
- Security issues
- Performance problems
- Best practices violations

Be concise and actionable.
```

Usage: `/review src/auth.py`

## Variables

- `$ARGUMENTS` - Everything after the command name
- Files can be referenced and Claude will read them
