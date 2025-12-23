# Claude → Local LLM Workflow

Step-by-step guide for using Claude to seed a project, then using local LLMs to do work.

---

## Phase 1: Claude Session (Analyze & Seed)

Open Claude Code in your project:

```bash
cd ~/Code/rat-king
claude
```

### Step 1: Ask Claude to analyze

```
Analyze this codebase and give me:
1. A summary of what this project does
2. The main files and their purposes
3. Areas that need work (bugs, missing features, tech debt)
```

### Step 2: Ask Claude to create seeds

Pick ONE thing from the analysis and ask Claude to create a seed:

**For adding a feature:**
```
Create a code skeleton with TODOs for [feature X].
Put it in seeds/skeletons/[name].py
Make the TODOs specific and numbered.
```

**For fixing/improving code:**
```
Create a checklist (YES/NO questions only) for reviewing [file or component].
Put it in seeds/checklists/[name].md
```

**For writing tests:**
```
Write failing tests for [function/class].
Put them in seeds/tests/test_[name].py
I'll have a local LLM write the implementation.
```

**For refactoring:**
```
Create a transformation spec for converting [X to Y].
Include rules, example input, and example output.
Put it in seeds/transforms/[name].md
```

### Step 3: Claude creates the file

Claude writes the seed file directly into your project. Verify it looks right.

### Step 4: Commit the seed (optional)

```bash
git add seeds/
git commit -m "Add seed: [description]"
```

---

## Phase 2: Local LLM Session (Execute)

Use clood to have local LLMs implement the seeds:

### Option A: Direct ask

```bash
# Read the seed and pipe to clood
cat seeds/skeletons/my-feature.py | clood ask "Implement all TODOs in this code"

# Or with context
clood ask "Implement all TODOs" --context seeds/skeletons/my-feature.py
```

### Option B: Interactive chat

```bash
clood chat
# Then paste the seed and prompt
```

### Prompting tips for local LLMs

**For skeletons:**
```
Implement this code by filling in all the TODO comments.
Output only the completed code, no explanations.
```

**For checklists:**
```
Answer each question with only YES or NO.
One answer per line.
```

**For tests:**
```
Write the implementation that makes all these tests pass.
Output only the code, no explanations.
```

**For transforms:**
```
Apply these rules to convert the code.
Output only the converted code.
```

### Step 3: Review and fix

Local LLM output needs review:
- Check for obvious errors
- Run tests/linter
- Fix minor issues manually

---

## Quick Reference

| I want to... | Claude creates... | clood prompt |
|--------------|-------------------|--------------|
| Add feature | Skeleton with TODOs | "Implement the TODOs" |
| Review code | YES/NO checklist | "Answer YES or NO only" |
| Write tests | Test file | "Write code to pass these tests" |
| Refactor | Transform spec + rules | "Apply these transformation rules" |
| Add docs | Docstring template | "Add docstrings following this format" |
| Add types | Type annotation examples | "Add type annotations like these" |

---

## Directory Structure in Your Project

```
my-project/
├── src/                    # Your actual code
├── tests/
├── seeds/                  # Add this directory
│   ├── skeletons/         # Code with TODOs
│   ├── checklists/        # YES/NO review lists
│   ├── tests/             # Test-first specs
│   └── transforms/        # Refactoring rules
└── ...
```

The `seeds/` directory is for Claude ↔ local LLM handoff. You can gitignore it or keep it for reference.

---

## Tips

1. **One seed = one task** - Don't combine multiple features
2. **Be specific in TODOs** - "Validate email with regex" not "Validate input"
3. **Include examples** - Local LLMs learn from examples
4. **Review everything** - Local LLMs make mistakes
5. **Iterate** - If output is wrong, refine the seed and try again
