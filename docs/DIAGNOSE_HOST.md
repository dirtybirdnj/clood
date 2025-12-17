# Host Connectivity Diagnostic Guide

Instructions for an agent running on a host (e.g., ubuntu25) to diagnose why it appears offline to clood.

---

## Quick Diagnostic Commands

Run these and report the output:

```bash
# 1. Is Ollama running?
systemctl status ollama
# or
pgrep -a ollama

# 2. What port is Ollama listening on?
ss -tlnp | grep 11434
# or
netstat -tlnp | grep 11434

# 3. Is Ollama bound to all interfaces or just localhost?
curl http://localhost:11434/api/version
curl http://0.0.0.0:11434/api/version
curl http://$(hostname -I | awk '{print $1}'):11434/api/version

# 4. What's this machine's IP?
hostname -I
ip addr show | grep "inet "

# 5. Is the firewall blocking port 11434?
sudo ufw status
sudo iptables -L -n | grep 11434

# 6. Can we reach the outside? (test from host perspective)
ping -c 2 192.168.4.1  # typical gateway
```

---

## Common Issues and Fixes

### Issue: Ollama bound to localhost only

**Symptom:** `curl http://localhost:11434` works, but `curl http://<lan-ip>:11434` fails.

**Check:**
```bash
ss -tlnp | grep 11434
# If you see 127.0.0.1:11434, it's localhost-only
# Should show 0.0.0.0:11434 for all interfaces
```

**Fix:**
```bash
# Edit Ollama service to bind to all interfaces
sudo systemctl edit ollama

# Add these lines:
[Service]
Environment="OLLAMA_HOST=0.0.0.0"

# Then restart
sudo systemctl daemon-reload
sudo systemctl restart ollama
```

**Alternative fix (environment variable):**
```bash
# Add to ~/.bashrc or /etc/environment
export OLLAMA_HOST=0.0.0.0

# Restart Ollama
sudo systemctl restart ollama
```

### Issue: Firewall blocking port

**Check:**
```bash
sudo ufw status verbose
```

**Fix:**
```bash
sudo ufw allow 11434/tcp
sudo ufw reload
```

### Issue: Ollama not running

**Check:**
```bash
systemctl status ollama
```

**Fix:**
```bash
sudo systemctl start ollama
sudo systemctl enable ollama  # auto-start on boot
```

### Issue: Wrong IP address

**Check:**
```bash
hostname -I
# Compare with what clood is trying to reach (192.168.4.63)
```

**Fix:** Update clood config or `/etc/hosts` on the calling machine.

---

## Full Diagnostic Report Template

Run this script and share the output:

```bash
echo "=== OLLAMA DIAGNOSTIC REPORT ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo ""

echo "=== IP Addresses ==="
hostname -I
echo ""

echo "=== Ollama Process ==="
pgrep -a ollama || echo "NOT RUNNING"
echo ""

echo "=== Ollama Service Status ==="
systemctl status ollama --no-pager 2>/dev/null || echo "No systemd service"
echo ""

echo "=== Port 11434 Listening ==="
ss -tlnp | grep 11434 || echo "NOT LISTENING"
echo ""

echo "=== Localhost Test ==="
curl -s http://localhost:11434/api/version || echo "FAILED"
echo ""

echo "=== LAN IP Test ==="
LAN_IP=$(hostname -I | awk '{print $1}')
curl -s http://${LAN_IP}:11434/api/version || echo "FAILED"
echo ""

echo "=== Firewall Status ==="
sudo ufw status 2>/dev/null || echo "UFW not installed"
echo ""

echo "=== Ollama Environment ==="
systemctl show ollama --property=Environment 2>/dev/null || echo "N/A"
echo ""

echo "=== END REPORT ==="
```

---

## Expected Good Output

```
=== Port 11434 Listening ===
LISTEN 0 4096 0.0.0.0:11434 0.0.0.0:* users:(("ollama",pid=1234,fd=3))

=== Localhost Test ===
{"version":"0.5.4"}

=== LAN IP Test ===
{"version":"0.5.4"}
```

Key indicators:
- `0.0.0.0:11434` (not `127.0.0.1:11434`)
- Both localhost and LAN IP tests return version JSON

---

## Diagnosing from Outside (Driver Machine)

Run these from the machine trying to reach the host (e.g., mac-mini or MacBook):

```bash
# Direct connectivity test
curl http://192.168.4.63:11434/api/version

# With timeout (don't wait forever)
curl --connect-timeout 5 http://192.168.4.63:11434/api/version

# Check if host is reachable at all
ping -c 2 192.168.4.63

# Check if port is open (nc/netcat)
nc -zv 192.168.4.63 11434

# What clood sees
./clood hosts
```

### Interpreting Results

| Outside Test | Inside Test | Diagnosis |
|--------------|-------------|-----------|
| Timeout | localhost works | **Binding issue** - Ollama on 127.0.0.1 only |
| Connection refused | localhost works | **Firewall** - port 11434 blocked |
| Timeout | ping fails | **Network issue** - host unreachable |
| Works | Works | **Clood bug** - check clood config/code |

### Quick Fix Checklist

If outside test times out but inside localhost works:

```bash
# On the host (ubuntu25), run:
sudo systemctl edit ollama

# Add exactly these lines:
[Service]
Environment="OLLAMA_HOST=0.0.0.0"

# Save, then:
sudo systemctl daemon-reload
sudo systemctl restart ollama

# Verify from outside:
curl http://192.168.4.63:11434/api/version
```

---

## The Iron Keep (ubuntu25)

The legendary ubuntu25, known as **The Iron Keep**, stands as the primary GPU workhorse in the clood garden. When this beast goes silent, the garden feels the absence.

### Iron Keep Quick Check

```bash
# From your driver machine, verify the Keep is awake:
curl --connect-timeout 5 http://ubuntu25:11434/api/version

# SSH in and check GPU health:
ssh ubuntu25 "nvidia-smi --query-gpu=name,memory.total,memory.used,temperature.gpu --format=csv"

# Check what models the Keep has loaded:
ssh ubuntu25 "ollama list"

# Full diagnostic dump:
ssh ubuntu25 "ollama ps && echo '---' && df -h /home/ollama-models"
```

### Haikus from the Iron Keep

*When diagnosing connectivity issues, meditate on these truths:*

```
Port eleven four
Three four—gateway to wisdom
Is it listening?
```

```
Zero zero zero
Point zero—all interfaces
Not just localhost
```

```
The firewall sleeps
But does it dream of blocking?
Check ufw status
```

```
Iron Keep stands tall
GPU fans spin through the night
Inference awaits
```

```
Timeout means nothing
If the binding is too tight
Loosen to zero
```

---

*"Debug the garden, one node at a time."*
