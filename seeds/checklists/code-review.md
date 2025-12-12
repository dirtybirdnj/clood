# Code Review Checklist

Answer ONLY "YES" or "NO" for each question.
One answer per line, in order. No explanations.

---

Code to review:
```
{PASTE CODE HERE}
```

---

## Structure
1. Are all functions under 50 lines?
2. Are all files under 500 lines?
3. Is there only one class per file?
4. Are imports grouped (stdlib, third-party, local)?

## Naming
5. Are variable names descriptive (not single letters except loops)?
6. Are function names verbs or verb phrases?
7. Are class names nouns in PascalCase?
8. Are constants in UPPER_SNAKE_CASE?

## Error Handling
9. Are all exceptions caught specifically (not bare except)?
10. Are error messages descriptive?
11. Are errors logged before re-raising?

## Documentation
12. Do all public functions have docstrings?
13. Are complex algorithms commented?
14. Is there a module-level docstring?

## Testing
15. Are there corresponding test files?
16. Do function names describe what is tested?

## Security
17. Is user input validated?
18. Are secrets externalized (not hardcoded)?
19. Is SQL parameterized?
20. Are file operations path-safe?
