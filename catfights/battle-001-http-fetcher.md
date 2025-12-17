# Kitchen Stadium - Battle 1: HTTP Fetcher

*December 17, 2025 - Evening*

---

## The Challenge

Implement `internal/sources/fetcher.go` for Chimborazo with:
- Fetcher struct with CacheDir and http.Client
- NewFetcher, CachePath (SHA256), IsCached
- Fetch with FetchOption pattern
- Clear, ClearAll
- Proper error wrapping

---

## Scoreboard

| Cat | Model | Time | Lines | Status |
|-----|-------|------|-------|--------|
| 1 | qwen2.5-coder:3b (Siamese) | 20s | 154 | Missing context import |
| 2 | mistral:7b (Tabby) | 72s | 220 | Hallucinated go-kit, tar |
| 3 | starcoder2:7b (Bengal) | 3s | 0 | EMPTY - No output! |
| 4 | deepseek-r1:14b (Serval) | 256s | 157 | Missing hex import |
| 5 | codellama:13b (Snow Leopard) | 112s | 108 | Wrong package name |
| 6 | llama3.1:8b (Caracal) | 58s | 206 | Wrong FetchOption pattern |
| 7 | deepseek-coder:6.7b (Persian) | 27s | 121 | Cleanest approach! |
| 8 | llama3-groq-tool-use:8b (Ocelot) | 28s | 119 | Invalid package syntax |
| 9 | tinyllama (Kitten) | 8s | 45 | Pure hallucination |

---

## Commendations

### 🥇 Persian (deepseek-coder:6.7b)
Cleanest approach in just 27 seconds. Good FetchOption pattern with internal options struct. Proper error handling. Minor issues: weird WithForce signature, context leak in WithTimeout.

### 🥈 Siamese (qwen2.5-coder:3b)
Fastest with real output (20s). Interface-based FetchOption is interesting but over-complex. Missing context import was its downfall.

### 🥉 Caracal (llama3.1:8b)
Most prolific (206 lines) in moderate time (58s). Added file extension on cache paths (.bin). Wrong FetchOption pattern though (struct instead of functional).

---

## MXC Hall of Shame 🎭

**Bengal (starcoder2:7b):**
> *silent stare* - The Bengal has no moves! GET IT TOGETHER!

**Chaos Kitten (tinyllama):**
> `FeTCHER`, `FeTCCHOption` - The Kitten speaks in a language known only to itself!

**Tabby (mistral:7b):**
> Imported `golang.org/x/crypto/tar` for an HTTP fetcher. *RIGHT YOU ARE, KEN!*

**Ocelot (llama3-groq-tool-use:8b):**
> `package internal/sources` - That's not how Go packages work!

---

## Iron Chef Claude's Synthesis

Key improvements over individual outputs:
1. **Atomic writes** - Write to `.tmp`, then rename (prevents corrupt cache)
2. **Proper cleanup** - Remove partial downloads on failure
3. **Status code check** - Don't cache error responses
4. **Cache metadata** - Return file mod time and size for cached results
5. **30s default timeout** - Reasonable default
6. **All standard library** - No hallucinated dependencies!

The synthesized implementation is saved at `/tmp/fetcher_synthesis.go`.

---

## Lessons Learned

1. **Sequential > Parallel** - Each cat needs full resources
2. **Small can win** - Persian (6.7b) beat House Lion (14b)
3. **Speed ≠ Quality** - Bengal was fastest but empty
4. **Hallucinations are consistent** - Same cats hallucinate repeatedly
5. **Synthesis beats competition** - Best result combines multiple approaches

---

## Haiku

```
Nine cats circle prey
Only one speaks plainly true
Iron Chef combines
```

---

*Next Battle: Recipe Parser*
