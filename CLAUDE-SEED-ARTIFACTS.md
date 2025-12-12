# Claude Seed Artifacts

How to use Claude (expensive, smart) to create artifacts that local LLMs (free, limited) can execute.

## The Problem

Small local models (7B and under) can't do:
- Open-ended analysis ("review this codebase")
- Planning ("design a system for X")
- Complex reasoning
- Tool use / agentic behavior

They output garbage like `{"name": "analyze_code", "arguments": {...}}` because they've seen tool patterns in training but can't actually reason.

## The Solution

Claude does the thinking. Local LLMs do the typing.

Create **seed artifacts** - structured documents that constrain the local LLM to tasks it can actually handle:
- Fill-in-the-blank code
- Explicit step-by-step instructions
- Yes/No checklists
- Template completion

---

## Artifact Types

### 1. Code Skeletons with TODOs

Claude creates the structure, local LLM fills in implementation.

**Claude creates:**
```python
# file: utils/image_processor.py
from PIL import Image
from pathlib import Path

def load_and_resize(path: str, size: tuple[int, int] = (512, 512)) -> Image.Image:
    """Load an image and resize it.

    Args:
        path: Path to image file
        size: Target (width, height), default 512x512

    Returns:
        Resized PIL Image in RGB mode
    """
    # TODO: Open image from path using Image.open()
    # TODO: Convert to RGB using .convert('RGB')
    # TODO: Resize using .resize() with LANCZOS resampling
    # TODO: Return the processed image
    pass


def batch_process(directory: str, output_dir: str) -> list[str]:
    """Process all images in a directory.

    Args:
        directory: Input directory path
        output_dir: Where to save processed images

    Returns:
        List of output file paths
    """
    # TODO: Create output_dir if it doesn't exist (use Path.mkdir)
    # TODO: Iterate over .jpg, .png, .webp files in directory
    # TODO: Call load_and_resize() on each
    # TODO: Save to output_dir with same filename
    # TODO: Collect and return list of saved paths
    pass
```

**Prompt for local LLM:**
```
Implement this function following the TODO comments exactly.
Only output the completed function, no explanations.
```

---

### 2. Transformation Tasks

Specific input→output transformations with examples.

**Claude creates:**
```markdown
# Task: Convert pytest tests to unittest

## Rules:
- `assert x == y` → `self.assertEqual(x, y)`
- `assert x` → `self.assertTrue(x)`
- `assert not x` → `self.assertFalse(x)`
- `assert x in y` → `self.assertIn(x, y)`
- `with pytest.raises(X):` → `with self.assertRaises(X):`
- Function `test_foo()` → Method `def test_foo(self):`
- Add `import unittest` at top
- Wrap in `class TestX(unittest.TestCase):`

## Example Input:
def test_addition():
    assert add(2, 2) == 4
    assert add(0, 0) == 0

## Example Output:
import unittest

class TestAddition(unittest.TestCase):
    def test_addition(self):
        self.assertEqual(add(2, 2), 4)
        self.assertEqual(add(0, 0), 0)

## Now convert this:
{paste code here}
```

---

### 3. Checklists (Yes/No Only)

Local LLMs can handle binary decisions on specific criteria.

**Claude creates:**
```markdown
# Security Review Checklist

Review the following code and answer ONLY "YES" or "NO" for each question.
Do not explain, do not add commentary. Just YES or NO per line.

Code:
{paste code}

Questions:
1. Are there any hardcoded passwords, API keys, or secrets?
2. Is user input used directly in SQL queries without parameterization?
3. Is user input used directly in shell commands?
4. Are there any eval() or exec() calls with user data?
5. Is sensitive data logged or printed?
6. Are file paths constructed from user input without validation?
7. Is there any use of pickle.loads() on untrusted data?
8. Are cryptographic operations using weak algorithms (MD5, SHA1)?
```

---

### 4. Fill-in-the-Blank Templates

Local LLMs excel at completion when heavily constrained.

**Claude creates:**
```markdown
# Generate API endpoint

Complete ONLY the blanks marked ___. Do not add anything else.

```python
from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

router = APIRouter()

class CreateUserRequest(BaseModel):
    username: str
    email: str

class UserResponse(BaseModel):
    id: int
    username: str
    email: str

@router.post("/users", response_model=___)
async def create_user(request: ___):
    # Validate email format
    if not ___.email.endswith("@"):
        raise HTTPException(status_code=___, detail="Invalid email")

    # Create user (mock)
    user_id = ___  # generate random int

    return ___(
        id=user_id,
        username=request.___,
        email=request.___
    )
```
```

---

### 5. Test-First Implementation

Claude writes tests, local LLM writes code to pass them.

**Claude creates:**
```python
# tests/test_calculator.py
# DO NOT MODIFY THESE TESTS

import pytest
from calculator import Calculator

def test_add():
    calc = Calculator()
    assert calc.add(2, 3) == 5
    assert calc.add(-1, 1) == 0
    assert calc.add(0, 0) == 0

def test_divide():
    calc = Calculator()
    assert calc.divide(10, 2) == 5
    assert calc.divide(7, 2) == 3.5

def test_divide_by_zero():
    calc = Calculator()
    with pytest.raises(ValueError, match="Cannot divide by zero"):
        calc.divide(1, 0)

def test_memory():
    calc = Calculator()
    calc.add(5, 3)
    assert calc.last_result == 8
    calc.divide(calc.last_result, 2)
    assert calc.last_result == 4
```

**Prompt for local LLM:**
```
Write calculator.py with a Calculator class that passes all these tests.
Include only the code, no explanations.
```

---

### 6. Docstring/Comment Generation

Small models are decent at this specific task.

**Prompt template:**
```markdown
Add a docstring to this function. Use Google style.
Include: summary, Args, Returns, Raises (if applicable), Example.

Function:
```python
def fetch_user_data(user_id, include_private=False, timeout=30):
    response = requests.get(
        f"{API_URL}/users/{user_id}",
        params={"private": include_private},
        timeout=timeout
    )
    response.raise_for_status()
    return response.json()
```

Output only the function with the docstring added.
```

---

### 7. Regex/Pattern Generation

Constrained generation task - works well.

**Prompt template:**
```markdown
Write a Python regex pattern for the following requirement.
Output ONLY the raw pattern string, nothing else.

Requirement: Match email addresses that:
- Start with alphanumeric characters
- May contain dots or underscores in the local part
- Have an @ symbol
- Domain has at least one dot
- TLD is 2-6 characters

Output format: r"your_pattern_here"
```

---

## Workflow

### In Claude (this CLI):

1. Analyze the codebase / understand the problem
2. Create a seed artifact (skeleton, checklist, template)
3. Save to `seeds/` directory or drop zone
4. Commit and push to repo

### In Crush (local LLM):

1. Pull repo or copy seed artifact
2. Paste the artifact as the prompt
3. Paste the target code if needed
4. Get output (may need to clean up)

### Example Session:

```bash
# Claude session - create the seed
claude
> analyze src/api/ and create a TODO skeleton for adding rate limiting

# Crush session - implement it
crush
> [paste the skeleton Claude created]
> implement this following the TODOs exactly
```

---

## Anti-Patterns (Don't Do This)

❌ "Review this codebase and suggest improvements"
❌ "Design a new feature for X"
❌ "What's wrong with this architecture?"
❌ "Refactor this to be better"
❌ "Write comprehensive tests"

These require reasoning. Small models will hallucinate or output tool-call JSON.

---

## Good Patterns

✅ "Fill in the TODO comments in this code"
✅ "Answer YES or NO to each question"
✅ "Convert this using the rules provided"
✅ "Complete only the blank spaces"
✅ "Add a docstring following this exact format"
✅ "Write code that passes these specific tests"

These are constrained completion tasks. Small models handle them.

---

## Directory Structure

```
clood/
├── seeds/                    # Claude-generated seed artifacts
│   ├── skeletons/           # Code skeletons with TODOs
│   ├── checklists/          # Yes/No review checklists
│   ├── transforms/          # Input→Output transformation specs
│   └── tests/               # Test-first implementation specs
│
├── prompts/
│   └── local-llm/           # Prompt templates for local models
│
└── drop-zone/               # Quick file staging
```

---

## Tips

1. **Be explicit** - Local LLMs follow instructions literally
2. **Provide examples** - Show exact input/output format
3. **Constrain output** - "Output ONLY the code, no explanations"
4. **One task at a time** - Don't chain multiple steps
5. **Validate output** - Small models make mistakes, review before using
