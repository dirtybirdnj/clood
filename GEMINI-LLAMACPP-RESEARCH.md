# llama.cpp Integration Research

> Research notes from exploring llama.cpp as a backend for clood.
> The goal: unlock the "raw metal" performance and control that Ollama abstracts away.

---

## The Discovery

Ollama is a wrapper around llama.cpp. Every model you've downloaded with `ollama pull` is a standard `.gguf` file, just renamed to its SHA256 hash and buried in Ollama's directory structure.

This means: **zero storage waste** - you can use the same files with both Ollama and llama.cpp.

---

## Zero-Waste Symlink Strategy

### Step 1: Find the GGUF blob

```bash
# Get the actual file path for any Ollama model
ollama show qwen2.5-coder:7b --modelfile | grep "FROM"
```

Output:
```
FROM /Users/yourname/.ollama/models/blobs/sha256-4b95d9703332...
```

That `sha256-...` file IS the GGUF. No conversion needed.

### Step 2: Symlink to llama.cpp

```bash
cd ~/Code/llama.cpp/models
ln -s ~/.ollama/models/blobs/sha256-4b95... qwen2.5-coder-7b.gguf
```

Now you have both:
- Ollama serving the model via its API
- llama.cpp ready to run the same file directly

---

## Ollama vs llama.cpp: What You Gain

| Feature | Ollama | llama.cpp (raw) |
|---------|--------|-----------------|
| **VRAM Control** | Auto-managed (conservative) | Manual (`-ngl 99` = all layers on GPU) |
| **Context Size** | Fixed by `num_ctx` | Dynamic (`-c 8192`, `-c 32768`) |
| **Flash Attention** | Sometimes enabled | Explicit (`-fa` or `--flash-attn`) |
| **KV Cache Quant** | Hidden | Manual (`-ctk q8_0` = bigger contexts in same VRAM) |
| **Samplers** | Simplified | Granular (Min-P, DRY, Mirostat tuning) |
| **Structured Output** | Basic JSON mode | **GBNF Grammars** (guaranteed structure) |
| **Overhead** | HTTP + management layer | Direct inference |

---

## The Killer Feature: GBNF Grammars

This is the game-changer for tool-use and structured output.

**Ollama's JSON Mode**: "Please output valid JSON" (model sometimes hallucinates)

**llama.cpp GBNF**: A formal grammar that *physically constrains* the model's output. It literally cannot emit tokens that violate the grammar.

### Example: Extracting a file path

With Ollama:
```
Model output: "Sure! The file you're looking for is: /home/user/code.go"
You: *write regex to parse this*
```

With GBNF grammar:
```
root ::= [a-zA-Z0-9/_\-\.]+
```
```
Model output: /home/user/code.go
You: *done*
```

The model cannot add conversational filler. It can only emit path characters.

---

## Running llama-server (The "Naked" Backend)

```bash
./llama-server \
  -m models/qwen2.5-coder-7b.gguf \
  --port 8080 \
  --ctx-size 8192 \
  --n-gpu-layers 99 \
  --flash-attn \
  --host 0.0.0.0
```

This exposes an **OpenAI-compatible API** on port 8080. Clood can talk to it without code changes.

---

## Implications for Clood

### 1. Hybrid Garden Architecture

The config could support both backends:

```yaml
hosts:
  # Ollama - friendly, managed, good defaults
  - name: ubuntu25-ollama
    url: http://192.168.4.63:11434
    type: ollama
    priority: 2

  # llama.cpp - raw, fast, full control
  - name: ubuntu25-raw
    url: http://192.168.4.63:8080
    type: llamacpp
    priority: 1
```

### 2. Guaranteed Structured Output

For MCP tools that need JSON (like `clood_analyze`, `clood_extract`), GBNF grammars eliminate parsing failures. The model *cannot* hallucinate invalid JSON.

This could be the missing piece for reliable tool-use with local models.

### 3. Performance Gains

Flash Attention + explicit GPU layer control = measurably faster inference. Use `clood catfight` to benchmark:

```bash
# Start llama-server with flash attention
./llama-server -m models/mistral.gguf --port 8080 -fa -ngl 99

# Compare same model, both backends
clood catfight "Write quicksort in Go" \
  --hosts "ubuntu25-ollama,ubuntu25-raw"
```

### 4. The "clood" Experience

If "clood" was about:
- **Reliable tool use** → GBNF grammars solve this
- **Fast responses** → Flash Attention + direct inference
- **Large context** → Manual `-c 32768` with KV cache quantization
- **Predictable output** → Grammar constraints

Then yes, llama.cpp might be the path to that experience with local models.

---

## Next Steps

1. **Build llama.cpp** on ubuntu25 (the GPU rig)
2. **Symlink existing Ollama models** (zero new downloads)
3. **Add `type: llamacpp` support** to clood's host config
4. **Implement GBNF grammar injection** for structured tool responses
5. **Benchmark** Ollama vs llama-server with catfight

---

## The Lore

*"In the beginning, there was Ollama, and it was good. It brought the tokens to the people. But as the Saga grew, the tokens became heavy. The Garden needed more speed. The user descended into the mines of ggml and forged a direct link to the metal. Now, the Ubuntu server doesn't just 'run models'; it emits the hum of raw computation, unshielded and pure."*

---

*Research conducted December 2024*
