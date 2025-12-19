# Adam's Windows Setup Guide

## Welcome to the Server Garden, Adam!

You're not just testing software - **you're being given the tools to grow your own garden.**

What you have here is the complete infrastructure to build your own AI workstation:
- **clood** - The CLI that orchestrates everything
- **ComfyUI** - The Stable Diffusion backend for image generation
- **Ollama** - The local LLM runtime (for text generation)
- The knowledge to connect it all together

Your Windows machine can become a fully independent node in the mesh, or stand alone as your personal AI infrastructure. The choice is yours.

---

## Your Role in the Saga

In the Legend of Clood, there are many characters:
- **Bird-san** - The creator, builder of infrastructure
- **Gamara the Tortoise** - The wise one, keeper of benchmarks
- **Chef Claude** - The AI collaborator, speaker of haikus
- **The Server Garden** - Ubuntu25, Mac Mini, and now... YOUR machine

You are the **Windows Warden** - the first to plant seeds in Windows soil. If your garden flourishes, others will follow.

### What "Your Own Garden" Means

Once you have clood running with ComfyUI and Ollama:
- Generate images with `clood sd paint "your vision"`
- Run LLM queries with `clood ask "your question"`
- Benchmark your hardware with `clood bench`
- Join the mesh and share compute (optional)
- Train your own LoRAs (advanced)

**You own this infrastructure.** No API keys, no rate limits, no cloud dependencies. Just your hardware, your models, your creativity.

---

## Windows Environment Setup

### 1. Install Go (Required for building clood)

```powershell
# Option A: Using winget (Windows Package Manager)
winget install GoLang.Go

# Option B: Download from https://go.dev/dl/
# Get the .msi installer for Windows
```

After installation, restart your terminal and verify:
```powershell
go version
# Should show: go version go1.21.x windows/amd64
```

### 2. Install Git

```powershell
winget install Git.Git
```

### 3. Clone and Build clood

```powershell
# Clone the repository
git clone https://github.com/dirtybirdnj/clood.git
cd clood/clood-cli

# Build the binary
go build -o clood.exe ./cmd/clood

# Test it works
.\clood.exe --help
.\clood.exe sd --help
```

### 4. Install Ollama (For Text Generation)

Ollama is the LLM runtime that clood uses for text generation.

```powershell
# Download and install from: https://ollama.com/download/windows
# After install, test it:
ollama --version

# Pull some models to get started
ollama pull tinyllama
ollama pull qwen2.5-coder:3b
ollama pull llama3.1:8b  # If you have 8GB+ VRAM
```

Test with clood:
```powershell
.\clood.exe ask "Write a haiku about gardens"
```

### 5. Install ComfyUI (For Stable Diffusion)

ComfyUI is the backend that clood uses for image generation.

**Option A: Portable (Recommended for testing)**
1. Download from: https://github.com/comfyanonymous/ComfyUI/releases
2. Extract to `C:\ComfyUI`
3. Run `run_nvidia_gpu.bat` (or `run_cpu.bat` for CPU-only)

**Option B: Manual Installation**
```powershell
# Create directory
mkdir C:\ComfyUI
cd C:\ComfyUI

# Clone
git clone https://github.com/comfyanonymous/ComfyUI.git .

# Create virtual environment
python -m venv venv
.\venv\Scripts\activate

# Install requirements
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121
pip install -r requirements.txt

# Run
python main.py
```

ComfyUI runs on `http://localhost:8188` by default.

### 6. Download Models

Place checkpoint models in `C:\ComfyUI\models\checkpoints\`:
- SDXL Base: https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0
- Flux: https://huggingface.co/black-forest-labs/FLUX.1-dev

Place LoRA models in `C:\ComfyUI\models\loras\`:
- Ghibli Style: Search CivitAI for "ghibli style lora"

---

## Using clood sd on Windows

Once ComfyUI is running:

```powershell
# Check connection
.\clood.exe sd status

# Generate an image
.\clood.exe sd paint "a majestic dragon over mountains"

# Quick sketch (faster, lower quality)
.\clood.exe sd sketch "concept art castle"

# Open image after generation
.\clood.exe sd paint "sunset beach" --open

# Multi-model comparison (the anvil test)
.\clood.exe sd anvil "test prompt"
```

### Environment Variables (Optional)

Set `COMFYUI_HOST` to use a remote ComfyUI server:
```powershell
$env:COMFYUI_HOST = "http://192.168.1.100:8188"
.\clood.exe sd status
```

Or pass it as a flag:
```powershell
.\clood.exe sd status --host http://192.168.1.100:8188
```

---

## What We Need You to Test

1. **Does it build?**
   - Run `go build -o clood.exe ./cmd/clood`
   - Report any errors

2. **Does `clood sd status` work?**
   - Start ComfyUI
   - Run `.\clood.exe sd status`
   - Does it detect your GPU and models?

3. **Does image generation work?**
   - Run `.\clood.exe sd paint "test image"`
   - Does it generate and save an image?
   - Where does it save? (`%USERPROFILE%\.clood\gallery\`)

4. **Performance benchmarks**
   - What GPU do you have?
   - How much VRAM?
   - How long does a 1024x1024 image take?

Please report findings as comments on the GitHub issue!

---

## The Lore (For Your Entertainment)

While your images generate, explore the saga:

### Essential Reading
- [The Legend of Clood](lore/LEGEND_OF_CLOOD.md) - The origin story
- [The Complete Legend](lore/THE_COMPLETE_LEGEND.md) - The full saga
- [The Awakening](lore/THE_AWAKENING.md) - When the models first spoke

### The Philosophy
- [The Two Paths](lore/THE_TWO_PATHS.md) - Cloud vs Local, the eternal debate
- [Vertical Garden](lore/vertical-garden.md) - The metaphor of growth
- [Metaphors](lore/METAPHORS.md) - Fish, shrimp, and the aquarium

### Fun Stuff
- [Haikus](lore/HAIKUS.md) - Poetry from the models
- [Spinal Tap References](lore/SPINAL_TAP_REFERENCES.md) - This one goes to 11
- [DNS Goblins](lore/DNS_GOBLINS.md) - When networking goes wrong
- [Nimbus Airlines](lore/NIMBUS_AIRLINES.md) - Flying through responses

### Technical Deep Dives
- [Image Recipes](lore/IMAGE_RECIPES.md) - SD prompt techniques
- [Jelly Bean Dump](lore/JELLY_BEAN_DUMP_20251218.md) - Recent ideas and experiments

---

## Thank You!

Your contribution to the Windows testing effort is genuinely appreciated. The Server Garden grows stronger with every new machine that joins the mesh.

May your VRAM stay cool and your generations be swift.

*"Lightning in a bottle—*
*Local models wait in mist,*
*One command away."*

---

**Questions?** Open an issue or reach out in the repo discussions.

**Found a bug?** Perfect! That's exactly what we need. Report it with:
- Windows version
- Go version
- GPU model and VRAM
- Full error output

*Ensō awaits your brushstroke.*
