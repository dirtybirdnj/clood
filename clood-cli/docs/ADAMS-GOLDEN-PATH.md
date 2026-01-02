# Adam's Golden Path

Step-by-step guide to get clood running on your Windows workstation.

---

## What You Need First

Before starting, make sure you have:

1. **Ollama installed** - Download from [ollama.ai/download](https://ollama.ai/download)
2. **Docker Desktop** - See detailed instructions below
3. **NVIDIA drivers** - Your GPU should be working (you probably already have this)
4. **At least one AI model** - We'll do this in Step 2

---

## Installing Docker Desktop (Required for Web UI)

Docker runs the Open WebUI chat interface. Here's how to set it up on Windows:

### Step 1: Download Docker Desktop

1. Go to: https://www.docker.com/products/docker-desktop/
2. Click **"Download for Windows"**
3. Run the installer (`Docker Desktop Installer.exe`)

### Step 2: Install with WSL 2 Backend

During installation:
1. Check **"Use WSL 2 instead of Hyper-V"** (recommended)
2. Check **"Add shortcut to desktop"**
3. Click **Install**
4. **Restart your computer** when prompted

### Step 3: First Run Setup

1. Open **Docker Desktop** from Start menu
2. Accept the license agreement
3. Wait for Docker to start (whale icon in system tray will stop animating)
4. You may see a tutorial - you can skip it

### Step 4: Verify Docker Works

Open PowerShell and run:

```powershell
docker --version
docker ps
```

If both commands work without errors, Docker is ready.

### Troubleshooting Docker

**"WSL 2 installation is incomplete"**
1. Open PowerShell as Administrator
2. Run: `wsl --install`
3. Restart your computer
4. Try Docker Desktop again

**"Virtualization must be enabled"**
1. Restart your computer
2. Enter BIOS (usually F2, F12, or Del during boot)
3. Find "Virtualization" or "VT-x" setting and enable it
4. Save and exit BIOS

**Docker Desktop won't start**
1. Make sure no other VM software is running (VirtualBox, VMware)
2. Try running as Administrator
3. Uninstall and reinstall Docker Desktop

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

## Step 5: Start the Web UI (Easiest Way)

The easiest way to use clood is with the web interface:

```powershell
.\clood.exe ui
```

This:
1. Starts the clood proxy (routes to your Ollama)
2. Starts Open WebUI (chat interface in Docker)
3. Opens your browser automatically

**Requirements for `clood ui`:**
- Docker Desktop must be running
- Ollama must be running

To stop everything:
```powershell
.\clood.exe ui --stop
```

### Adding Code Reader Tool (Optional)

Want the AI to read and analyze your code projects? Add the Code Reader tool:

1. Open http://localhost:3000 in your browser
2. Create an account (first account becomes admin)
3. Click your profile → **Workspace** → **Tools** → **+** (create)
4. Paste the contents of `skills/open-webui/code-reader.py` from the clood repo
5. Click **Save**

Now you can ask things like:
```
Show me the tree of my-project
Read the file my-project/README.md
Search for "TODO" in my-project
```

**Note:** Your code directory needs to be mounted in Docker. Edit `infrastructure/docker-compose.yml` and change the volume mount to your code location.

---

## Step 6: Try CLI Commands

For quick terminal access, run some commands:

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
| `.\clood.exe ui` | Start web chat interface (easiest) |
| `.\clood.exe ui --stop` | Stop web chat interface |
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

### "clood ui" not working / Docker issues

1. Make sure Docker Desktop is running (look for the whale icon in system tray)
2. Open Docker Desktop and wait for it to say "Running"
3. Try: `docker ps` - if this works, Docker is ready
4. If Docker won't start, you may need to enable virtualization in BIOS

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

You have ONE machine. The dev environment for clood has THREE machines networked together (the "server garden").

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
