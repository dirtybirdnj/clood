# SSH Setup for Multi-Machine Workflow

Passwordless SSH setup between clood machines for file transfer and remote access.

## Machines

| Host | Hostname | IP | Purpose |
|------|----------|-----|---------|
| ubuntu25 | ubuntu25 | 192.168.4.63 | Workstation (RX 590, Ollama server) |
| macbook-air | Mathews-MacBook-Air | DHCP | M4 MacBook Air |
| mac-mini | Mats-Mac-mini.local | 192.168.4.41 | M4 Mac Mini (clood primary) |

---

## Ubuntu25 Setup (Server)

### Install SSH Server

```bash
sudo apt install openssh-server
sudo systemctl enable --now ssh
sudo systemctl status ssh

# Allow through firewall
sudo ufw allow ssh
```

### Generate Project-Specific Key

```bash
ssh-keygen -t ed25519 -C "clood-ubuntu25" -f ~/.ssh/clood_ed25519
```

### SSH Config

Add to `~/.ssh/config`:

```
Host macbook-air
    HostName <MAC_IP>
    User mgilbert
    IdentityFile ~/.ssh/clood_ed25519

Host mac-mini
    HostName <MINI_IP>
    User mgilbert
    IdentityFile ~/.ssh/clood_ed25519
```

---

## macOS Setup (MacBook Air / Mac Mini)

### Generate Project-Specific Key

```bash
ssh-keygen -t ed25519 -C "clood-macbook-air" -f ~/.ssh/clood_ed25519
```

### SSH Config

Add to `~/.ssh/config`:

```
Host ubuntu25
    HostName 192.168.4.63
    User mgilbert
    IdentityFile ~/.ssh/clood_ed25519
```

---

## Exchange Keys

After SSH is running on all machines, exchange public keys:

### From Mac → ubuntu25

```bash
ssh-copy-id -i ~/.ssh/clood_ed25519.pub mgilbert@192.168.4.63
```

### From ubuntu25 → Mac

First get Mac's IP:
```bash
# On Mac
ipconfig getifaddr en0
```

Then from ubuntu25:
```bash
ssh-copy-id -i ~/.ssh/clood_ed25519.pub mgilbert@<MAC_IP>
```

---

## Test Connections

```bash
# From Mac
ssh ubuntu25

# From ubuntu25
ssh macbook-air
```

---

## Common Uses

### Sync Ollama Models (faster than downloading)

From Mac, pull models from ubuntu25:
```bash
rsync -av --progress ubuntu25:~/.ollama/models/ ~/.ollama/models/
```

### Remote Ollama Commands

```bash
ssh ubuntu25 "ollama list"
ssh ubuntu25 "ollama ps"
```

### Copy Files

```bash
# Copy file to ubuntu25
scp myfile.txt ubuntu25:~/Code/

# Copy from ubuntu25
scp ubuntu25:~/Code/output.txt .
```

---

## Troubleshooting

### Connection Refused

SSH server not running:
```bash
sudo systemctl start ssh
sudo systemctl status ssh
```

Firewall blocking:
```bash
sudo ufw allow ssh
sudo ufw status
```

### Permission Denied (publickey)

Key not copied:
```bash
ssh-copy-id -i ~/.ssh/clood_ed25519.pub user@host
```

Wrong permissions:
```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/clood_ed25519
chmod 644 ~/.ssh/clood_ed25519.pub
```

### Host Key Verification Failed

Remote host key changed (reinstall, new machine):
```bash
ssh-keygen -R 192.168.4.63
```

---

## Disk Cleanup (ubuntu25)

If disk is full, common cleanup commands:

```bash
# Overview
df -h

# Find big directories
du -h --max-depth=1 / 2>/dev/null | sort -hr | head -20

# Docker cleanup (removes unused images/containers)
docker system prune -a

# Apt cache
sudo apt clean
sudo apt autoremove

# Journal logs
sudo journalctl --vacuum-size=100M

# Check ollama models
du -sh ~/.ollama

# Check snap packages
du -sh /var/lib/snapd
```
