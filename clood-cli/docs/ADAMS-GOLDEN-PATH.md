# Adam's Golden Path

Complete from-scratch guide to get clood running on your Windows workstation. No developer experience required.

---

## Overview: What We're Installing

| Software | What It Does | Required? |
|----------|--------------|-----------|
| Ollama | Runs AI models locally on your GPU | Yes |
| Docker Desktop | Runs the chat web interface | Yes |
| clood | Connects everything together | Yes |
| Git | Downloads source code (only if building yourself) | Optional |

---

## Part 1: Install Ollama (The AI Engine)

Ollama runs AI models directly on your NVIDIA GPU.

### Step 1: Download Ollama

1. Go to: https://ollama.ai/download
2. Click **"Download for Windows"**
3. Run the installer
4. Click through the installation (all defaults are fine)

### Step 2: Verify Ollama is Running

After installation, Ollama starts automatically. Look for the llama icon in your system tray (bottom-right of screen, near the clock).

### Step 3: Open PowerShell

PowerShell is Windows' command-line tool. Here's how to open it:

1. Press **Windows key + X**
2. Click **"Windows PowerShell"** or **"Terminal"**

Or:
1. Press **Windows key**
2. Type **"powershell"**
3. Press **Enter**

### Step 4: Download Your First AI Model

In PowerShell, type this and press Enter:

```powershell
ollama pull qwen2.5-coder:7b
```

This downloads a 4GB AI model. Wait for it to complete (might take 5-10 minutes depending on your internet).

### Step 5: Test Ollama

```powershell
ollama run qwen2.5-coder:7b "Say hello"
```

If you see a response, Ollama is working! Press `Ctrl+D` to exit.

---

## Part 2: Install Docker Desktop (For Web Interface)

Docker runs the chat interface in your browser.

### Step 1: Download Docker Desktop

1. Go to: https://www.docker.com/products/docker-desktop/
2. Click **"Download for Windows"**
3. Run the installer (`Docker Desktop Installer.exe`)

### Step 2: Install with WSL 2

During installation:
1. **Check** "Use WSL 2 instead of Hyper-V" (recommended)
2. **Check** "Add shortcut to desktop"
3. Click **Install**
4. **Restart your computer** when prompted

### Step 3: First Run

After restart:
1. Open **Docker Desktop** from your desktop or Start menu
2. Accept the license agreement
3. Skip any tutorials
4. Wait for the whale icon in system tray to stop animating

### Step 4: Verify Docker Works

Open PowerShell and run:

```powershell
docker --version
```

If you see a version number (like `Docker version 24.0.6`), Docker is ready.

### Troubleshooting Docker

**"WSL 2 installation is incomplete"**
1. Open PowerShell **as Administrator** (right-click → Run as Administrator)
2. Run: `wsl --install`
3. Restart your computer
4. Open Docker Desktop again

**"Virtualization must be enabled"**
1. Restart your computer
2. During boot, press F2, F12, or Del to enter BIOS (varies by computer)
3. Find "Virtualization", "VT-x", or "SVM" setting
4. Enable it
5. Save and exit BIOS (usually F10)

**"Docker Desktop won't start"**
1. Close any VirtualBox or VMware if running
2. Right-click Docker Desktop → Run as Administrator
3. If still failing, uninstall and reinstall Docker Desktop

---

## Part 3: Get clood

### Option A: Download Pre-built (Recommended for Beginners)

1. Go to: https://github.com/dirtybirdnj/clood/releases
2. Download `clood-windows-amd64.exe`
3. Create a folder: `C:\Tools`
4. Move the downloaded file to `C:\Tools`
5. Rename it from `clood-windows-amd64.exe` to `clood.exe`

Now navigate to it in PowerShell:

```powershell
cd C:\Tools
.\clood.exe --version
```

### Option B: Build From Source (For Developers)

This requires Git and Go. Skip this if you used Option A.

**Install Git:**
```powershell
winget install Git.Git
```
Close and reopen PowerShell.

**Install Go:**
```powershell
winget install GoLang.Go
```
Close and reopen PowerShell.

**Build clood:**
```powershell
git clone https://github.com/dirtybirdnj/clood.git
cd clood\clood-cli
go build -o clood.exe .\cmd\clood
.\clood.exe --version
```

---

## Part 4: Download More AI Models

The more models you have, the more options. With your powerful GPU, you can run big ones:

```powershell
# Fast small model
ollama pull qwen2.5-coder:3b

# Larger reasoning model
ollama pull llama3.1:8b

# Tool-capable model (can use tools in Open WebUI)
ollama pull llama3-groq-tool-use:8b

# Big coding model (if you have 24GB+ VRAM)
ollama pull qwen2.5-coder:14b
```

---

## Part 5: Start the Web Interface

This is the easy part. Make sure:
- Docker Desktop is running (whale icon in system tray)
- Ollama is running (llama icon in system tray)

Then run:

```powershell
cd C:\Tools
.\clood.exe ui
```

This will:
1. Start the clood proxy
2. Start Open WebUI in Docker
3. Open your browser to http://localhost:3000

**First time setup in the browser:**
1. Click **"Sign Up"**
2. Create an account (this is local, just for you)
3. Start chatting!

To stop everything later:
```powershell
.\clood.exe ui --stop
```

---

## Part 6: Using the Chat Interface

### Selecting a Model

1. In the chat, click the model name at the top
2. Select from your downloaded models
3. Start typing!

### Recommended Models for Different Tasks

| Task | Model | Why |
|------|-------|-----|
| Quick questions | qwen2.5-coder:3b | Fast responses |
| Coding help | qwen2.5-coder:7b | Good balance |
| Complex reasoning | llama3.1:8b | Smarter |
| Using tools | llama3-groq-tool-use:8b | Can call functions |

---

## Common Commands Cheat Sheet

Run these from PowerShell in your clood folder:

| Command | What It Does |
|---------|--------------|
| `.\clood.exe ui` | Start web chat interface |
| `.\clood.exe ui --stop` | Stop web chat interface |
| `.\clood.exe ask "question"` | Quick question from terminal |
| `.\clood.exe models` | List your downloaded models |
| `.\clood.exe system` | Show your hardware info |
| `.\clood.exe doctor` | Check if everything is working |
| `.\clood.exe --help` | Show all available commands |

---

## Troubleshooting

### "clood.exe is not recognized"

You need to be in the folder where clood.exe lives:

```powershell
cd C:\Tools
.\clood.exe --help
```

### "Cannot connect to Ollama"

1. Look in your system tray for the llama icon
2. If not there, open Ollama from Start menu
3. Test with: `ollama list`

### "No GPU detected"

1. Run: `nvidia-smi`
2. If that fails, update NVIDIA drivers from https://www.nvidia.com/drivers
3. Restart your computer

### "clood ui" says Docker not running

1. Look for the whale icon in system tray
2. If not there, open Docker Desktop from Start menu
3. Wait for it to fully start (whale stops animating)
4. Try again

### Browser shows error or blank page

1. Wait 30 seconds (Open WebUI takes time to start)
2. Refresh the page
3. Try http://localhost:3000 manually

### Something else is broken

Run this and send me the output:

```powershell
.\clood.exe doctor -v
```

---

## Your Hardware Advantage

With your powerful GPU, you can run models most people can't:

| Model | Size | Your Experience |
|-------|------|-----------------|
| qwen2.5-coder:3b | 2GB | Instant |
| qwen2.5-coder:7b | 4GB | Fast |
| llama3.1:8b | 5GB | Fast |
| deepseek-r1:14b | 9GB | Smooth |
| qwen2.5-coder:14b | 9GB | Smooth |
| codestral:22b | 13GB | Doable |
| qwen2.5-coder:32b | 20GB | If you have VRAM |

Pull bigger models anytime:
```powershell
ollama pull codestral:22b
```

---

## Need Help?

1. Run `.\clood.exe doctor -v` and send me the output
2. Create an issue at: https://github.com/dirtybirdnj/clood/issues/new
3. Include:
   - What you tried
   - What error you saw
   - Output from `.\clood.exe doctor -v`

---

*Your machine. Your models. Your rules.*
