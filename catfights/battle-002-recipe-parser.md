# Kitchen Stadium - Battle 2: Recipe Parser

*December 17, 2025 - Night*

---

## The Challenge

Implement Recipe type definitions and YAML loader for Chimborazo:
- `internal/config/recipe.go` - Type definitions
- `internal/config/loader.go` - YAML loading with validation
- Use gopkg.in/yaml.v3

---

## Scoreboard

| Cat | Model | Time | Lines | Status |
|-----|-------|------|-------|--------|
| 🥇 | deepseek-coder:6.7b (Persian) | 20s | 74 | Clean, all YAML tags |
| 🥈 | mistral:7b (Tabby) | 21s | 76 | Redeemed! No hallucinations |
| 🥉 | qwen2.5-coder:3b (Siamese) | 12s | 84 | Fast but missing YAML tags |
| ⚠️ | llama3.1:8b (Caracal) | 27s | 112 | Circular import, syntax error |
| ⚠️ | deepseek-r1:14b (House Lion) | 146s | 75 | Syntax error, wrong field name |
| ⚠️ | codellama:13b (Snow Leopard) | 53s | 66 | Missing all YAML tags |

**Skipped:** Bengal (starcoder2:7b), Kitten (tinyllama) - MXC failures from Battle 1

---

## Commendations

### 🥇 Persian (deepseek-coder:6.7b) - BACK TO BACK CHAMPION!
Two battles in a row! Clean output in 20 seconds. All YAML tags present. Proper error wrapping. Simple but complete.

### 🥈 Tabby (mistral:7b) - REDEMPTION ARC!
After hallucinating `golang.org/x/crypto/tar` in Battle 1, the Tabby returns clean and focused. 21 seconds, no hallucinated dependencies. Uses deprecated ioutil but functionally correct.

### 🥉 Siamese (qwen2.5-coder:3b) - SPEED KING
Fastest real output at 12 seconds. Better validation logic than Persian. Missing YAML tags and uses undefined `readFile` function.

---

## MXC Moments 🎭

**Caracal (llama3.1:8b):**
> Circular import: `github.com/dirtybirdnj/chimborazo/internal/config` inside config package!
> Hallucinated logrus. Syntax error `.Bounds` with leading period.

**House Lion (deepseek-r1:14b):**
> 146 seconds for... a syntax error? `(Source string ...)` with parentheses!
> Also named a field `Output` instead of `Source` in Layer struct.

**Snow Leopard (codellama:13b):**
> "I'll define types without YAML tags" - Snow Leopard, probably
> Missing imports for ioutil and yaml despite using them.

---

## Iron Chef Claude's Synthesis

Key improvements:
1. **All YAML tags** - Learned from Persian
2. **Comprehensive validation** - Learned from Siamese
3. **Modern os.ReadFile** - Avoiding deprecated ioutil
4. **omitempty tags** - For optional fields (Operations, Style, Filter)
5. **Descriptive errors** - Layer index and name in validation messages
6. **Path in error** - Include filename when read fails

The synthesized files:
- `/tmp/recipe_synthesis.go` - Type definitions
- `/tmp/loader_synthesis.go` - YAML loader with validation

---

## Lessons Learned

1. **Consistency matters** - Persian won both battles
2. **Size ≠ speed** - House Lion (14b) was slowest AND had errors
3. **Redemption is possible** - Tabby cleaned up its act
4. **YAML tags are essential** - Half the cats forgot them
5. **Circular imports are common** - Watch for self-referencing packages

---

## Haiku

```
Persian wins again
Small and swift defeats the large
Tags make YAML sing
```

---

*Next Battle: TBD - Perhaps geometry operations?*
