# Last Session - 2025-12-12 (Afternoon)

## What We Did

### 1. Created CLAUDE.md Agent Guidelines
- Added rule: **Always ask before installing software** (brew, apt, pip, etc.)
- Added benchmarking guidelines with creative elements (haiku/shanty)
- Location: `/Users/mgilbert/Code/clood/CLAUDE.md`

### 2. SSH Multi-Machine Setup
- Created comprehensive SSH setup guide: `infrastructure/SSH-SETUP.md`
- Generated project-specific SSH key on MacBook Air: `~/.ssh/clood_ed25519`
- Configured `~/.ssh/config` with ubuntu25 host (IP: **192.168.4.63** - changed from .62 due to DHCP)
- SSH now working between MacBook Air and ubuntu25

### 3. ASCII Login Prompt Guide
- Created `infrastructure/ascii-login-prompt.md`
- Covers dynamic MOTD, system stats, Champ ASCII art
- Includes Ubuntu Pro ad removal commands

### 4. Ollama Model Sync (IN PROGRESS)
- Rsync running: `rsync -av --progress ubuntu25:/home/ollama-models/ ~/.ollama/models/`
- Models on ubuntu25 were moved to `/home/ollama-models/` (disk space issue)
- Transfer speed: ~30MB/s over local network
- Total size: ~20GB
- **Status: Still running when session paused**

### 5. Sea Shanty: The Ballad of Handy's Lunch
Created a Lake Champlain winter sea shanty about:
- Sailing to Handy's for a chili dog
- Passing the Four Brothers islands
- Getting sunk by Champ

## Weather Check
Burlington VT (Dec 12, 2025): 27°F, flurries, west winds 13mph, 1-3" snow tonight

## Pending Tasks

1. **Wait for rsync to complete** - check with `ollama list` on MacBook Air
2. **Run benchmarks on M4 MacBook Air** (32GB):
   ```bash
   ollama run tinyllama "Write hello world in Python" --verbose 2>&1 | grep "eval rate"
   ollama run qwen2.5-coder:3b "Write fizzbuzz in Python" --verbose 2>&1 | grep "eval rate"
   ```
3. **Add benchmark results to ollama-tuning.md** with sea shanty

## ubuntu25 Tasks (Disk Was Full)

SSH server installed, but ollama needs to point to moved models:

```bash
# Edit override.conf
sudo nano /etc/systemd/system/ollama.service.d/override.conf

# Add this line:
Environment="OLLAMA_MODELS=/home/ollama-models"

# Reload
sudo systemctl daemon-reload
sudo systemctl restart ollama
ollama list
```

Also clean up stray SSH key in models folder:
```bash
sudo rm /home/ollama-models/id_ed25519 /home/ollama-models/id_ed25519.pub
```

## IP Addresses (Current)

| Machine | IP | Notes |
|---------|-----|-------|
| ubuntu25 | 192.168.4.63 | Changed from .62 (DHCP) |
| MacBook Air | DHCP | M4, 32GB |

## Files Changed This Session

- `CLAUDE.md` - NEW - Agent guidelines
- `infrastructure/SSH-SETUP.md` - NEW - Multi-machine SSH setup
- `infrastructure/ascii-login-prompt.md` - NEW - Login customization
- `~/.ssh/config` - Updated with ubuntu25 host
- `~/.ssh/clood_ed25519` - NEW - Project SSH key

## Background Task Running

```
Task ID: bcfc186
Command: rsync models from ubuntu25
Status: Running (~56% when last checked)
Output: /tmp/claude/tasks/bcfc186.output
```

To check if done:
```bash
ollama list  # Should show models when sync complete
```

## Resume Commands

When you come back:
```bash
# Check if sync finished
ollama list

# If models present, run benchmarks
ollama serve &  # Start ollama if not running
ollama run tinyllama "Hello" --verbose 2>&1 | grep "eval rate"
ollama run qwen2.5-coder:3b "Write fizzbuzz" --verbose 2>&1 | grep "eval rate"
```

## The Ballad of Handy's Lunch (For Reference)

*(A Lake Champlain Winter Shanty)*

**Chorus:**
*Heave ho! The lake is cold!*
*Champ lurks below where waters hold!*
*A chili dog waits on the Burlington shore*
*But we may never taste one anymore!*

Key verse:
> Past the Four Brothers islands we steered through the snow
> The captain cried "Faster! To Handy's we go!"
