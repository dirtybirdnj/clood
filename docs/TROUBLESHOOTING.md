# Troubleshooting Guide

Quick reference for common clood failure modes and their solutions.

---

## Quick Diagnostics

```bash
# Check overall system health
clood health

# Check which hosts are online
clood hosts

# Check available models
clood models

# Verbose preflight check
clood preflight
```

---

## Connectivity Issues

### Ollama Not Responding

**Symptom:** `clood hosts` shows host as offline, or requests timeout.

**Check:**
```bash
# Is Ollama running?
curl http://localhost:11434/api/version

# For remote hosts
curl http://<host-ip>:11434/api/version
```

**Solutions:**
1. Start Ollama: `ollama serve` or `systemctl start ollama`
2. Check binding (see [DIAGNOSE_HOST.md](DIAGNOSE_HOST.md) for details)
3. Verify firewall allows port 11434

### SSH Tunnel Issues

**Symptom:** Remote host works via direct IP but not through tunnel.

**Check:**
```bash
# Verify tunnel is active
lsof -i :11434

# Test tunnel endpoint
curl http://localhost:11434/api/version
```

**Solutions:**
1. Restart tunnel: `ssh -L 11434:localhost:11434 user@host`
2. Check for port conflicts (another Ollama instance?)
3. Verify remote Ollama is bound to localhost

### Request Timeouts

**Symptom:** Requests hang then fail with timeout errors.

**Causes:**
- Model loading (first request after cold start)
- Large model on slow hardware
- Network latency for remote hosts

**Solutions:**
1. Wait for initial model load (can take 30-60s)
2. Use smaller model: `--model qwen2.5-coder:3b`
3. Increase timeout in config

---

## Model Issues

### Model Not Found

**Symptom:** `Error: model 'xyz' not found`

**Check:**
```bash
# List models on specific host
clood models --host <hostname>

# Or via ollama directly
ollama list
```

**Solutions:**
1. Pull the model: `ollama pull <model-name>`
2. Check model name spelling (case-sensitive)
3. Update config to use available model

### Out of Memory (OOM)

**Symptom:** Inference starts then crashes, or system becomes unresponsive.

**Signs:**
```bash
# Check system logs
dmesg | grep -i "out of memory"
journalctl -u ollama | tail -20
```

**Solutions:**
1. Use smaller model (3B instead of 7B, etc.)
2. Close other applications
3. Add swap space (temporary fix)
4. Route to more capable host: `clood ask --host <bigger-host> "query"`

### Slow Inference

**Symptom:** Model responds but takes very long.

**Check:**
```bash
# Quick benchmark
clood bench --quick
```

**Causes:**
- CPU inference (no GPU acceleration)
- Model too large for available VRAM
- Thermal throttling

**Solutions:**
1. Check GPU detection: `clood system`
2. Use smaller model for speed
3. Ensure adequate cooling

---

## Configuration Issues

### Config File Not Found

**Symptom:** `Error: config file not found`

**Location:** `~/.clood/config.yaml`

**Solution:**
```bash
# Initialize default config
clood init

# Or create manually
mkdir -p ~/.clood
cat > ~/.clood/config.yaml << 'EOF'
hosts:
  - name: local
    url: http://localhost:11434
    type: local
EOF
```

### Invalid YAML Syntax

**Symptom:** `Error: yaml: line X: ...`

**Check:**
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('$HOME/.clood/config.yaml'))"
```

**Common Issues:**
- Mixed tabs and spaces (use spaces only)
- Missing quotes around URLs
- Incorrect indentation

### Host Configuration Conflicts

**Symptom:** Wrong host selected, or host priority unexpected.

**Check:**
```bash
# Show merged config
clood config show
```

**Tips:**
- Hosts are tried in order listed
- Use `--host` flag to force specific host
- Check tier configuration for auto-routing

---

## MCP Server Issues

### SSE Connection Drops

**Symptom:** Claude Code loses connection to clood MCP server.

**Check:**
```bash
# Is MCP server running?
pgrep -f "clood mcp"

# Check Claude Code logs
cat ~/Library/Logs/Claude/mcp*.log 2>/dev/null
```

**Solutions:**
1. Restart MCP server
2. Check `~/.claude.json` for correct stdio config
3. Increase connection timeout in Claude Code settings

### Tool Registration Failures

**Symptom:** MCP tools not appearing in Claude Code.

**Check:**
```bash
# Test MCP server directly
echo '{"jsonrpc":"2.0","method":"initialize","id":1}' | clood mcp
```

**Solutions:**
1. Verify clood binary is in PATH
2. Check for JSON parsing errors in output
3. Restart Claude Code

---

## Catfight/Triage Issues

### GitHub Rate Limiting

**Symptom:** `Error: API rate limit exceeded`

**Check:**
```bash
gh api rate_limit
```

**Solutions:**
1. Wait for rate limit reset (shown in error)
2. Use `--delay` flag between operations
3. Authenticate with `gh auth login` for higher limits

### Model Timeout During Comparison

**Symptom:** Catfight hangs or times out mid-battle.

**Solutions:**
1. Use smaller models: `--models "qwen2.5-coder:3b,tinyllama"`
2. Shorter prompts
3. Check host health during battle: `clood hosts`

### Empty Responses

**Symptom:** Model returns empty or very short response.

**Causes:**
- Prompt too long (context overflow)
- Model confusion
- Incomplete model download

**Solutions:**
1. Verify model: `ollama show <model>`
2. Test with simple prompt first
3. Re-pull model: `ollama pull <model>`

---

## Getting Help

If none of the above resolves your issue:

1. **Check logs:** `~/.clood/logs/` (if logging enabled)
2. **Verbose mode:** Add `--verbose` or `-v` to commands
3. **File an issue:** https://github.com/dirtybirdnj/clood/issues

Include:
- clood version (`clood --version`)
- OS and architecture
- Relevant command and full error output
- Config (sanitize any credentials)

---

*When in doubt, `clood health` is your friend.*
