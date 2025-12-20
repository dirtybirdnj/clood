# Catfight Model Rankings

*Based on overnight triage analysis of 33 issues*

---

## Performance Tiers

### Tier 1: Production Ready (Proven Performers)

| Model | Cat Name | Strengths | Use Case |
|-------|----------|-----------|----------|
| qwen2.5-coder:3b | Siamese | 31 consistent scope estimates | Fast triage, code gen |
| qwen2.5-coder:7b | Maine Coon | Balanced speed/quality | General coding |
| deepseek-coder:6.7b | Persian | Accurate structured output | Catfight champion |
| mistral:7b | Tabby | Good all-rounder | General purpose |
| llama3.1:8b | Caracal | Strong reasoning | Complex analysis |

### Tier 2: Reliable (Good Output)

| Model | Observations |
|-------|--------------|
| yi-coder:1.5b | 9 good outputs, fast |
| falcon3:1b | 8 good outputs, tiny |
| stablelm2:1.6b | 8 good outputs |
| codellama:7b | Solid coding tasks |
| phi3:3.8b | Good reasoning |
| granite-code:3b | Reliable for code |

### Tier 3: Specialist

| Model | Use Case |
|-------|----------|
| tinyllama | Ultra-fast, simple tasks only |
| qwen2.5-coder:14b | When you need more power |

### ELIMINATED (Do Not Use)

| Model | Problem |
|-------|---------|
| **starcoder2:3b** | 24 EMPTY outputs in triage |
| **codegemma:2b** | Garbage/minimal output |

---

## Triage Statistics

### Coverage
- 33 of 74 open issues triaged (45%)
- Issues #68-100, #102 have catfight data

### Scope Consensus
```
M (Medium):  58%  ← Strong consensus
S (Small):   18%
L (Large):    8%
XS/XL:       10%
```

---

## Recommended Gauntlet Configuration

### Standard Triage (12 models)
```bash
MODELS="qwen2.5-coder:14b,llama3.1:8b,mistral:7b,deepseek-coder:6.7b,codellama:7b,phi3:3.8b,qwen2.5-coder:3b,granite-code:3b,stablelm2:1.6b,yi-coder:1.5b,falcon3:1b,tinyllama"
```

### Fast Triage (3 models)
```bash
MODELS="qwen2.5-coder:3b,mistral:7b,deepseek-coder:6.7b"
```

### Lightweight Triage (Small models only)
```bash
MODELS="qwen2.5-coder:3b,yi-coder:1.5b,falcon3:1b,tinyllama"
```

---

## Quality Indicators

Good triage output should include:
- Scope estimate (Size: XS/S/M/L/XL)
- Implementation plan
- Minimum 100 tokens
- Clear structure

---

## Catfight Command Reference

```bash
# Basic catfight with default cats
clood catfight "your prompt here"

# Custom models
clood catfight --models "qwen2.5-coder:3b,mistral:7b" "prompt"

# Live streaming (multiple models racing)
clood catfight-live "prompt"

# Save output
clood catfight --output-dir ./results "prompt"
```

---

*Analysis based on Kitchen Stadium battles, December 2024*
