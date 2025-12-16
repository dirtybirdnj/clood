# Clood Canaries & Golden Paths

Patterns for catching LLM failure modes early and producing predictable, high-quality output.

---

## Part 1: Canaries (Early Warning Signals)

### The Problem

LLMs fail silently by being "helpful" - filling gaps with assumptions rather than questioning premises. This creates drift from user intent that compounds over a session.

### Canary: File Not Found

**Anti-pattern observed:**
```
User: "walk me through the profiling script in docs/HARDWARE_FACTS.md"
LLM: *file not found* → "Let me design this feature for you!"
```

**What should happen:**
```
User: "walk me through the profiling script in docs/HARDWARE_FACTS.md"
LLM: *file not found* → "File not found. Should I git pull or check a different path?"
```

**Detection rules:**

| User Intent Keywords | Expectation | If Not Found |
|---------------------|-------------|--------------|
| review, walk through, show me, what's in, explain | EXISTS | STOP - verify path/pull |
| create, make, build, add, write | CREATE | Proceed |
| fix, update, modify, change | EXISTS | STOP - verify path/pull |

**Git freshness guard:**
```bash
# Check if remote has changes before declaring "not found"
git fetch --dry-run 2>&1 | grep -q "." && echo "Remote may have updates"

# Check if file exists on remote but not local
git ls-tree -r origin/main --name-only | grep -q "path/to/file" && echo "File exists on remote"
```

**CLAUDE.md rule:**
```markdown
## File Not Found Protocol

When a user references a specific file that doesn't exist:
1. NEVER assume it should be created
2. Check: Is the remote ahead? `git fetch --dry-run`
3. Ask: "File not found at X. Should I git pull, or is the path different?"
4. Only create if user explicitly says "create it" or "make a new one"
```

---

### Canary: Scope Creep

**Anti-pattern:**
```
User: "add a logout button"
LLM: *adds button, refactors auth system, adds session management, updates 15 files*
```

**Detection:** Count files touched vs. complexity of request. Simple request + many files = canary trigger.

**Guard:** Before multi-file changes, summarize scope and confirm.

---

### Canary: Assumption Cascade

**Anti-pattern:**
```
User: "why is the build failing?"
LLM: *assumes it's a type error* → *fixes wrong thing* → *breaks more* → *keeps "fixing"*
```

**Detection:** Multiple edit attempts without reading error output.

**Guard:** Read before write. Always. Especially errors.

---

## Part 2: Golden Paths (Reproducible Recipes)

Like civit.ai prompts that produce consistent image outputs, clood recipes produce consistent code patterns.

### Recipe Format

```yaml
name: "recipe-name"
description: "What this recipe produces"
prerequisites:
  - file: "path/to/required/file"
    must_exist: true
  - command: "go version"
    validates: "go1.21+"

prompt: |
  [Structured prompt that produces predictable output]

expected_output:
  files_created: ["list", "of", "files"]
  files_modified: ["existing", "files"]
  patterns_used: ["error-handling-v1", "config-loading"]

verification:
  - command: "go build ./..."
  - command: "go test ./..."
```

---

### Recipe: New CLI Command

**Name:** `new-cli-command`

**Prompt:**
```
Add a new command `clood {command_name}` that {description}.

Follow these patterns from the existing codebase:
1. Command file location: cmd/{command_name}.go
2. Use cobra command structure matching cmd/root.go
3. Add to root command in init()
4. Use the existing config loading pattern from cmd/config.go
5. Error handling: wrap errors with context, return to cobra

Do NOT:
- Create new packages for a single command
- Add dependencies not already in go.mod
- Modify unrelated files
- Add CLI flags without explicit request

Output:
1. The new command file
2. The modification to cmd/root.go (just the init addition)
```

**Expected output:**
- `cmd/{command_name}.go` - new file
- `cmd/root.go` - one line added to init()

---

### Recipe: Hardware Profile Script

**Name:** `hardware-profile`

**Prompt:**
```
I need to profile a new machine for the Server Garden.

Walk me through running the facts collection script:
1. Show me the one-liner for quick collection
2. Explain what each field means
3. Tell me what to paste back

Reference: clood-cli/docs/HARDWARE_FACTS.md

Do NOT:
- Recreate the script if it exists
- Run commands on my behalf without showing them first
- Assume the machine type (ask mac/linux)
```

---

### Recipe: Fix Build Error

**Name:** `fix-build-error`

**Prompt:**
```
The build is failing. Help me fix it.

Process:
1. Run the build command and capture full output
2. Read the ENTIRE error, not just the first line
3. Identify the root cause (not symptoms)
4. Show me the minimal fix
5. Verify the fix works

Do NOT:
- Assume the error type before reading it
- Fix multiple unrelated issues at once
- Refactor surrounding code
- Add error handling "while we're here"
```

---

### Recipe: Code Review

**Name:** `code-review`

**Prompt:**
```
Review {file_or_pr} for:
1. Bugs - logic errors, edge cases, null checks
2. Security - injection, auth, data exposure
3. Performance - obvious N+1, unnecessary allocations
4. Clarity - naming, structure, comments where non-obvious

Format:
- [BUG] line:XX - description
- [SEC] line:XX - description
- [PERF] line:XX - description
- [STYLE] line:XX - description (optional, only if egregious)

Do NOT:
- Suggest rewrites of working code
- Add type annotations or docstrings
- Recommend "improvements" beyond the review scope
- Say "looks good" without actually reviewing
```

---

## Part 3: Structural Guidelines

### The Chimbo Pattern

Predictable structure → predictable output. Every clood interaction should follow:

```
1. ORIENT   - What exists? Read before assuming.
2. PLAN     - What's the minimal change? State it.
3. EXECUTE  - Do exactly that, nothing more.
4. VERIFY   - Did it work? Prove it.
```

### Anti-Chimbo (What to Avoid)

```
1. ASSUME   - "This probably needs..."
2. EXPAND   - "While we're here, let's also..."
3. CREATE   - "I'll make a new abstraction for..."
4. HOPE     - "That should fix it" (without testing)
```

---

### File Organization Golden Path

```
project/
├── cmd/                    # CLI commands (one file per command)
├── internal/               # Private packages
│   ├── config/            # Configuration loading
│   ├── {domain}/          # Business logic by domain
│   └── {domain}/          # Not by technical layer
├── pkg/                    # Public packages (if any)
├── scripts/               # Shell scripts for ops
├── docs/                  # Documentation
│   └── recipes/           # Clood recipes (this pattern)
└── CLAUDE.md              # Agent instructions
```

### Naming Golden Path

```
Files:      lowercase-with-dashes.go
Packages:   lowercase, single word preferred
Functions:  VerbNoun (GetUser, ParseConfig, ValidateInput)
Variables:  camelCase, descriptive (userCount not uc)
Constants:  CamelCase or ALL_CAPS for true constants
```

---

## Part 4: Implementation in Clood CLI

### Pre-Flight Checks

Before agent execution, clood should:

```go
type PreFlightCheck struct {
    GitFresh     bool     // Is local up to date with remote?
    FilesExist   []string // Do referenced files exist?
    Intent       string   // review|create|modify|unknown
    ScopeEstimate int     // Estimated files to touch
}

func (c *PreFlightCheck) Warnings() []string {
    var warnings []string
    if !c.GitFresh {
        warnings = append(warnings, "Remote has changes - consider git pull")
    }
    for _, f := range c.FilesExist {
        if !fileExists(f) {
            warnings = append(warnings, fmt.Sprintf("Referenced file not found: %s", f))
        }
    }
    if c.Intent == "review" && c.ScopeEstimate == 0 {
        warnings = append(warnings, "Review requested but no files identified")
    }
    return warnings
}
```

### Recipe Loader

```go
type Recipe struct {
    Name         string
    Description  string
    Prompt       string
    Prerequisites []Prerequisite
    Verification  []string
}

func LoadRecipe(name string) (*Recipe, error) {
    path := filepath.Join("docs", "recipes", name+".yaml")
    // ...
}

func (r *Recipe) CheckPrerequisites() error {
    for _, p := range r.Prerequisites {
        if p.MustExist && !fileExists(p.File) {
            return fmt.Errorf("prerequisite not met: %s must exist", p.File)
        }
    }
    return nil
}
```

---

## Part 5: Prompt Library

Reusable prompt fragments for consistent behavior.

### Orient Block
```
Before making any changes:
1. Read all files I'll modify
2. Check git status for uncommitted changes
3. Verify I understand the current state
```

### Scope Block
```
Scope limit: Only modify files directly related to {task}.
Do not: refactor, add comments, improve error handling, or touch unrelated code.
```

### Verify Block
```
After changes:
1. Run the build
2. Run relevant tests
3. Show me the diff of what changed
4. Confirm the original request is satisfied
```

---

## Next Steps

1. [ ] Add canary checks to CLAUDE.md
2. [ ] Create `docs/recipes/` directory with initial recipes
3. [ ] Implement pre-flight checks in clood-cli
4. [ ] Build recipe loader and executor
5. [ ] Add `clood recipe list` and `clood recipe run {name}` commands
