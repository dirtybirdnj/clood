# ASCII Art Login Prompt for Ubuntu 25

Customize your SSH login with ASCII art and system info.

---

## Option 1: Static MOTD (Simple)

Replace the Message of the Day with custom ASCII art:

```bash
sudo nano /etc/motd
```

Paste your ASCII art, save, done. Shows after successful login.

---

## Option 2: Dynamic MOTD (Recommended)

Ubuntu uses scripts in `/etc/update-motd.d/` that run at login.

### Disable Default Clutter

```bash
# See what's there
ls -la /etc/update-motd.d/

# Disable the noisy ones
sudo chmod -x /etc/update-motd.d/10-help-text
sudo chmod -x /etc/update-motd.d/50-motd-news
sudo chmod -x /etc/update-motd.d/91-release-upgrade
```

### Add Custom ASCII Header

```bash
sudo nano /etc/update-motd.d/01-custom
```

```bash
#!/bin/bash

cat << 'EOF'

   _____ _      ____   ____  _____
  / ____| |    / __ \ / __ \|  __ \
 | |    | |   | |  | | |  | | |  | |
 | |    | |   | |  | | |  | | |  | |
 | |____| |___| |__| | |__| | |__| |
  \_____|______\____/ \____/|_____/

  ubuntu25 - RX 590 Vulkan + i7-8086K

EOF
```

Make it executable:
```bash
sudo chmod +x /etc/update-motd.d/01-custom
```

### Add System Stats

```bash
sudo nano /etc/update-motd.d/02-sysinfo
```

```bash
#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# System info
UPTIME=$(uptime -p)
LOAD=$(cat /proc/loadavg | awk '{print $1, $2, $3}')
MEMORY=$(free -h | awk '/^Mem:/ {print $3 "/" $2}')
DISK=$(df -h / | awk 'NR==2 {print $3 "/" $2 " (" $5 ")"}')

echo -e "${GREEN}Uptime:${NC}  $UPTIME"
echo -e "${GREEN}Load:${NC}    $LOAD"
echo -e "${GREEN}Memory:${NC}  $MEMORY"
echo -e "${GREEN}Disk:${NC}    $DISK"
echo ""
```

```bash
sudo chmod +x /etc/update-motd.d/02-sysinfo
```

### Test It

```bash
sudo run-parts /etc/update-motd.d/
```

---

## Option 3: Pre-Login Banner

Shows BEFORE password prompt (good for warnings/legal notices):

```bash
sudo nano /etc/issue.net
```

```
*******************************************
*  CLOOD WORKSTATION - Authorized Only    *
*******************************************

```

Enable in SSH config:
```bash
sudo nano /etc/ssh/sshd_config
```

Find and set:
```
Banner /etc/issue.net
```

Restart SSH:
```bash
sudo systemctl restart ssh
```

---

## ASCII Art Generators

- [patorjk.com/software/taag](https://patorjk.com/software/taag/) - Text to ASCII
- [asciiart.eu](https://www.asciiart.eu/) - ASCII art collection
- `figlet` - CLI tool: `sudo apt install figlet && figlet "CLOOD"`
- `toilet` - Colorful CLI: `sudo apt install toilet && toilet -f mono12 "CLOOD"`

---

## Example: Champ ASCII Art

```bash
sudo nano /etc/update-motd.d/01-champ
```

```bash
#!/bin/bash
cat << 'EOF'

         __
        / _)
 .-^^^-/ /
__/       /        CLOOD WORKSTATION
<__.|_|-|_|        ubuntu25 - Lake Champlain's Finest

EOF
```

```bash
sudo chmod +x /etc/update-motd.d/01-champ
```

---

## Disable Ubuntu Pro Ads

Ubuntu loves to advertise. Kill it:

```bash
sudo pro config set apt_news=false
sudo systemctl disable ubuntu-advantage
sudo chmod -x /etc/update-motd.d/88-esm-announce 2>/dev/null
sudo chmod -x /etc/update-motd.d/91-contract-ua-esm-status 2>/dev/null
```

---

## Full Example Setup

```bash
# Disable defaults
sudo chmod -x /etc/update-motd.d/10-help-text
sudo chmod -x /etc/update-motd.d/50-motd-news

# Create header
cat << 'SCRIPT' | sudo tee /etc/update-motd.d/01-clood
#!/bin/bash
cat << 'EOF'

   _____ _      ____   ____  _____
  / ____| |    / __ \ / __ \|  __ \
 | |    | |   | |  | | |  | | |  | |
 | |    | |   | |  | | |  | | |  | |
 | |____| |___| |__| | |__| | |__| |
  \_____|______\____/ \____/|_____/

EOF
SCRIPT
sudo chmod +x /etc/update-motd.d/01-clood

# Test
sudo run-parts /etc/update-motd.d/
```
