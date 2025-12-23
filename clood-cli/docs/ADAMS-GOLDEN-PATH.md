# Adam's Golden Path

Step-by-step guide to get clood running on your Windows workstation.

---

## What You Need First

Before starting, make sure you have:

1. **Ollama installed** - Download from [ollama.ai/download](https://ollama.ai/download)
2. **NVIDIA drivers** - Your GPU should be working (you probably already have this)
3. **At least one AI model** - We'll do this in Step 2

---

## Step 1: Get clood

**Option A: Download Pre-built (Easier)**

1. Go to: https://github.com/dirtybirdnj/clood/releases
2. Download `clood-windows-amd64.exe`
3. Rename it to `clood.exe`
4. Put it somewhere you'll remember (like `C:\Tools\clood.exe`)

**Option B: Build It Yourself**

Open PowerShell and run these commands one at a time:

```powershell
# Install Go (the programming language clood is written in)
winget install GoLang.Go
```

**IMPORTANT: Close PowerShell and open a new one after installing Go.**

Then continue:

```powershell
# Download the clood source code
git clone https://github.com/dirtybirdnj/clood.git

# Go into the folder
cd clood\clood-cli

# Build it
go build -o clood.exe .\cmd\clood

# Check it worked
.\clood.exe --version
```

---

## Step 2: Pull Some AI Models

Open a PowerShell window and run:

```powershell
# A small fast model (good for quick tests)
ollama pull qwen2.5-coder:3b

# A medium model (good balance)
ollama pull qwen2.5-coder:7b

# A larger model (slower but smarter)
ollama pull llama3.1:8b
```

Wait for each one to download. They're big files, might take a few minutes each.

---

## Step 3: Run Setup

Navigate to wherever you put clood.exe, then run:

```powershell
.\clood.exe setup
```

This will:
- Detect your hardware (GPU, memory, etc.)
- Find Ollama and your models
- Create a config file

---

## Step 4: Verify Everything Works

Run the doctor command:

```powershell
.\clood.exe doctor
```

This checks:
- Is your GPU detected?
- Is Ollama running?
- Are your models available?

If something's wrong, it tells you exactly how to fix it.

---

## Step 5: Try It Out

Now the fun part. Run some commands:

```powershell
# Ask a question
.\clood.exe ask "What is the capital of France?"

# See your system info
.\clood.exe system

# List your models
.\clood.exe models

# Compare two models on the same question
.\clood.exe catfight "Write a short poem about computers"
```

---

## Common Commands Cheat Sheet

| Command | What It Does |
|---------|--------------|
| `.\clood.exe ask "question"` | Ask the AI something |
| `.\clood.exe models` | List your downloaded models |
| `.\clood.exe system` | Show your hardware info |
| `.\clood.exe doctor` | Check if everything is working |
| `.\clood.exe catfight "prompt"` | Compare multiple models |
| `.\clood.exe --help` | Show all available commands |

---

## Troubleshooting

### "clood.exe is not recognized"

You need to run it from the folder where clood.exe lives:

```powershell
cd C:\Tools  # or wherever you put it
.\clood.exe --help
```

### "Cannot connect to Ollama"

1. Look in your system tray (bottom right of screen) for the Ollama icon
2. If it's not there, open Ollama from your Start menu
3. Try running: `ollama list` - if this works, Ollama is running

### "No GPU detected"

1. Open a command prompt and run: `nvidia-smi`
2. If that doesn't work, you need to install/update NVIDIA drivers
3. After updating drivers, restart your computer

### Something else is broken

Run this and send me the output:

```powershell
.\clood.exe doctor -v
```

---

## Your Hardware Advantage

With your beefy GPU (32GB+ VRAM), you can run bigger models than most people:

| Model | Difficulty for You |
|-------|-------------------|
| qwen2.5-coder:3b | Trivial |
| qwen2.5-coder:7b | Easy |
| llama3.1:8b | Easy |
| deepseek-r1:14b | Easy |
| qwen2.5-coder:14b | Easy |
| codestral:22b | Doable |

Want to try a bigger model? Just run:

```powershell
ollama pull codestral:22b
```

---

## What's Different for You

You have ONE machine. Bird-san has THREE machines networked together (the "server garden").

For you, everything is simpler:
- All commands run locally
- No need for network configuration
- `catfight` compares models on YOUR machine

Commands like `thunderdome`, `delegate`, and `hosts` exist for multi-machine setups. You can ignore them - your single powerful machine does everything locally.

---

## Report Back

After you get it working, let me know:

1. What GPU you have
2. How much VRAM
3. Which models you tried
4. Any issues you ran into

You can create an issue at: https://github.com/dirtybirdnj/clood/issues/new

---

*Your machine. Your models. Your rules.*
