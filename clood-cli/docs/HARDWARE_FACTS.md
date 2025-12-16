# Hardware Facts Collection

Collect hardware facts from machines in your Server Garden for clood configuration.

## Quick One-Liner

Run this on any Mac and paste the output:

```bash
echo "{\"hostname\":\"$(hostname)\",\"chip\":\"$(sysctl -n machdep.cpu.brand_string 2>/dev/null || system_profiler SPHardwareDataType | grep 'Chip' | awk -F: '{print $2}' | xargs)\",\"memory_gb\":$(( $(sysctl -n hw.memsize) / 1073741824 )),\"ollama\":$(curl -s http://localhost:11434/api/tags 2>/dev/null || echo 'null')}"
```

## Full Script

Save as `gather-facts.sh` and run: `bash gather-facts.sh > hostname-facts.json`

```bash
#!/bin/bash
# Hardware facts collection for clood Server Garden
# Usage: bash gather-facts.sh > $(hostname)-facts.json

set -e

echo "{"

# System identification
echo '  "hostname": "'$(hostname)'",'
echo '  "collected_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",'

# OS info
if [[ "$OSTYPE" == "darwin"* ]]; then
  echo '  "os": "'$(sw_vers -productName) $(sw_vers -productVersion)'",'

  # Try to get Apple Silicon chip name
  CHIP=$(system_profiler SPHardwareDataType 2>/dev/null | grep "Chip" | awk -F: '{print $2}' | xargs)
  if [ -z "$CHIP" ]; then
    CHIP=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Unknown")
  fi
  echo '  "chip": "'"$CHIP"'",'

elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
  echo '  "os": "'$(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)'",'
  echo '  "chip": "'$(cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d: -f2 | xargs)'",'

  # Check for NVIDIA GPU
  if command -v nvidia-smi &> /dev/null; then
    GPU=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
    VRAM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
    echo '  "gpu": "'"$GPU"'",'
    echo '  "vram": "'"$VRAM"'",'
  fi
fi

# Hardware specs
echo '  "cores": '$(nproc 2>/dev/null || sysctl -n hw.ncpu)','
if [[ "$OSTYPE" == "darwin"* ]]; then
  echo '  "memory_gb": '$(( $(sysctl -n hw.memsize) / 1073741824 ))','
else
  echo '  "memory_gb": '$(free -g | awk '/^Mem:/{print $2}')','
fi

# Ollama status
if curl -s --connect-timeout 2 http://localhost:11434/api/version > /dev/null 2>&1; then
  OLLAMA_VERSION=$(curl -s http://localhost:11434/api/version | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
  echo '  "ollama_running": true,'
  echo '  "ollama_version": "'"$OLLAMA_VERSION"'",'

  # Get models as JSON array
  MODELS=$(curl -s http://localhost:11434/api/tags | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    models = []
    for m in d.get('models', []):
        models.append({
            'name': m['name'],
            'size_gb': round(m.get('size', 0) / 1e9, 2),
            'parameter_size': m.get('details', {}).get('parameter_size', 'unknown'),
            'quantization': m.get('details', {}).get('quantization_level', 'unknown')
        })
    print(json.dumps(models, indent=4))
except:
    print('[]')
" 2>/dev/null || echo '[]')
  echo '  "models": '"$MODELS"
else
  echo '  "ollama_running": false,'
  echo '  "ollama_version": null,'
  echo '  "models": []'
fi

echo "}"
```

## Collected Facts

### ubuntu25 (collected 2025-12-16)

```json
{
  "hostname": "ubuntu25",
  "ollama_version": "0.13.2",
  "models": [
    {"name": "llama3.1:8b", "size_gb": 4.92, "parameter_size": "8.0B", "quantization": "Q4_K_M"},
    {"name": "qwen2.5-coder:7b", "size_gb": 4.68, "parameter_size": "7.6B", "quantization": "Q4_K_M"},
    {"name": "qwen2.5-coder:3b", "size_gb": 1.93, "parameter_size": "3.1B", "quantization": "Q4_K_M"},
    {"name": "tinyllama:latest", "size_gb": 0.64, "parameter_size": "1B", "quantization": "Q4_0"},
    {"name": "mistral:7b", "size_gb": 4.37, "parameter_size": "7.2B", "quantization": "Q4_K_M"},
    {"name": "llama3.2:3b", "size_gb": 2.02, "parameter_size": "3.2B", "quantization": "Q4_K_M"},
    {"name": "deepseek-coder:6.7b", "size_gb": 3.83, "parameter_size": "7B", "quantization": "Q4_0"},
    {"name": "llama3-groq-tool-use:8b", "size_gb": 4.66, "parameter_size": "8.0B", "quantization": "Q4_0"}
  ],
  "benchmarks": {
    "tinyllama:latest": {"gen_tok_per_sec": 198, "prompt_tok_per_sec": 925},
    "qwen2.5-coder:7b": {"gen_tok_per_sec": 35, "prompt_tok_per_sec": 123},
    "llama3.1:8b": {"gen_tok_per_sec": 34, "prompt_tok_per_sec": 97}
  }
}
```

### mac-mini (pending)

Upload your facts here after running the script.

```json
{
  "hostname": "mac-mini",
  "TODO": "Run gather-facts.sh and paste output here"
}
```

## Network Configuration

For clood to reach your hosts, ensure:

| Host | Expected URL | Notes |
|------|--------------|-------|
| ubuntu25 | http://192.168.4.63:11434 | GPU-accelerated |
| mac-mini | http://192.168.4.41:11434 | Apple Silicon |
| localhost | http://localhost:11434 | Optional local instance |
