# Adam's Golden Path

Windows setup guide for the Wojak power user. You have the hardware. You have the models. Now you need the ninjitsu.

## Prerequisites

- Windows 10/11
- Ollama installed and running
- Models already downloaded
- Command line access (PowerShell or CMD)

## Step 1: Install Go

```powershell
# Option A: winget (recommended)
winget install GoLang.Go

# Option B: Download from https://go.dev/dl/
# Get the .msi installer, run it, accept defaults
```

After install, **restart your terminal** (important!), then verify:

```powershell
go version
# Should show: go version go1.21.x windows/amd64
```

## Step 2: Clone and Build

```powershell
# Clone the repo
git clone https://github.com/dirtybirdnj/clood.git
cd clood/clood-cli

# Build it
go build -o clood.exe ./cmd/clood

# Verify it works
.\clood.exe --help
```

## Step 3: Configure Your Host

Create config file at `%USERPROFILE%\.config\clood\config.yaml`:

```yaml
hosts:
  - name: localhost
    url: http://localhost:11434
```

Test it:

```powershell
.\clood.exe hosts
# Should show localhost as online
```

## Step 4: First Catfight

```powershell
# List your models
.\clood.exe models

# Run a simple test
.\clood.exe catfight -m "qwen2.5-coder:7b" -m "llama3.1:8b" "Write fizzbuzz in Python"
```

## Reporting Your Hardware

**DO NOT commit hardware specs to the repo.**

Instead, create a GitHub issue:

1. Go to: https://github.com/dirtybirdnj/clood/issues/new
2. Title: `Hardware Profile: [your-hostname]`
3. Label: `hardware-profile`
4. Include:
   - GPU model and VRAM
   - RAM
   - CPU
   - Which models you have downloaded

This keeps your info in issues (searchable, commentable) not in the codebase.

### Quick Hardware Check

```powershell
# GPU info
nvidia-smi

# Or for the lazy
nvidia-smi --query-gpu=name,memory.total --format=csv
```

## Your Secret Weapon: VRAM

With 32GB+ VRAM, you can run models others can't:

| Model | VRAM Needed | Your Status |
|-------|-------------|-------------|
| qwen2.5-coder:7b | ~5GB | Easy |
| llama3.1:8b | ~6GB | Easy |
| qwen2.5-coder:14b | ~10GB | Easy |
| deepseek-coder:33b | ~20GB | Possible |
| llama3.1:70b-q4 | ~40GB | Maybe? |

You're in "big model" territory. Test the limits.

## Image Gen Workloads

Your VRAM makes you ideal for Stable Diffusion workloads. Coming soon:

- `clood sd` commands for image generation
- Model output catalog: same prompt, different models, compare results
- Batch processing for prompt variations

For now, focus on getting clood working with Ollama. Image gen integration is a future jelly bean.

## Model Output Catalog

The vision: run the same prompt against multiple models, save all outputs, compare.

```powershell
# Future command (not implemented yet)
clood catalog run "Explain recursion" --models qwen2.5-coder:7b,llama3.1:8b,deepseek-coder:6.7b

# Saves to:
# catalog/
#   2025-12-18_explain-recursion/
#     qwen2.5-coder-7b.md
#     llama3.1-8b.md
#     deepseek-coder-6.7b.md
#     summary.md
```

For now, use catfight and manually save interesting outputs.

## Quick Reference

```powershell
# Check what's running
.\clood.exe health

# See your models
.\clood.exe models

# System info
.\clood.exe system

# Run a catfight
.\clood.exe catfight -m model1 -m model2 "your prompt"

# Get JSON output (for scripting)
.\clood.exe models --json
```

## Troubleshooting

**"go: command not found"**
- Restart your terminal after installing Go
- Check PATH: `echo $env:PATH` should include Go

**"connection refused"**
- Is Ollama running? Check: `ollama list`
- Firewall blocking localhost:11434?

**Build errors**
- Run `go mod tidy` in the clood-cli directory
- Make sure you're in the right folder

## Next Steps

1. Get clood building and running
2. Create your hardware profile issue
3. Run some catfights
4. Report back what works, what doesn't
5. Prepare for image gen workloads

---

*The garden grows. Your VRAM is the soil.*
