# Repartitioning Guide for ubuntu25

This guide covers resizing partitions to give root more space from the oversized home partition.

## Current Partition Layout

```
Device           Size   Used  Avail  Mount
/dev/nvme0n1p1   779G   23G   717G   /home    <- Too big
/dev/nvme0n1p2   489M   6M    482M   /boot/efi
/dev/nvme0n1p3   92G    67G   20G    /        <- Too small
```

## Target Layout

```
Device           Size   Mount
/dev/nvme0n1p1   550G   /home   (shrink by ~230GB)
/dev/nvme0n1p3   320G   /       (grow by ~230GB)
```

This gives root plenty of room for Docker, Ollama models, system packages, etc.

---

## Requirements

- Ubuntu installer USB drive (works perfectly)
- ~30-60 minutes
- Backup of important data (recommended)

---

## Pre-Flight Checklist

Before rebooting:

```bash
# 1. Note current UUIDs (you may need these)
sudo blkid

# 2. Backup /etc/fstab
cp /etc/fstab ~/fstab.backup

# 3. Note partition layout
lsblk -f > ~/partitions.backup

# 4. Sync any unsaved work
sync
```

---

## Step-by-Step Process

### 1. Boot from Ubuntu USB

1. Insert Ubuntu installer USB
2. Reboot: `sudo reboot`
3. Press F12/F2/Del during POST to enter boot menu (depends on motherboard)
4. Select the USB drive
5. Choose **"Try Ubuntu"** (NOT Install)

### 2. Open GParted

Once in the live environment:

```bash
# GParted should be pre-installed
# If not:
sudo apt update && sudo apt install gparted
```

1. Click Activities → Search "GParted" → Open
2. Select your NVMe drive from dropdown (likely `/dev/nvme0n1`)

### 3. Unmount Partitions (if mounted)

GParted may auto-mount partitions. Right-click each and select "Unmount" if needed.

### 4. Shrink /home Partition

1. Right-click `/dev/nvme0n1p1` (the 779GB /home partition)
2. Select **Resize/Move**
3. Change "New size" to **550000** MiB (or drag the slider)
4. Click **Resize/Move**

**Important:** Shrink from the END of the partition (leave free space after it, not before).

### 5. Move/Resize Root Partition

This is the tricky part. The free space is now between /home and root.

**Option A: If root is AFTER home (nvme0n1p3 > nvme0n1p1)**

1. Right-click `/dev/nvme0n1p3` (root)
2. Select **Resize/Move**
3. Drag the LEFT edge to consume the free space
4. Click **Resize/Move**

**Option B: If partitions are in different order**

You may need to move partitions. GParted will show you visually.

### 6. Apply Changes

1. Click the green checkmark **Apply All Operations**
2. Confirm the warning
3. **Wait** - this can take 20-60 minutes for large partitions
4. Do NOT interrupt or power off

### 7. Verify

After completion:

1. Click **Refresh** in GParted
2. Verify new sizes look correct
3. Close GParted

### 8. Reboot

1. Remove USB drive
2. Reboot into your normal system

```bash
sudo reboot
```

---

## Post-Reboot Verification

```bash
# Check new sizes
df -h / /home

# Verify filesystems
sudo fsck -n /dev/nvme0n1p1
sudo fsck -n /dev/nvme0n1p3

# Check fstab still works (should be fine, uses UUIDs)
cat /etc/fstab
```

---

## Troubleshooting

### System won't boot after resize

1. Boot from USB again (Try Ubuntu)
2. Check if UUIDs changed: `sudo blkid`
3. Mount root and fix fstab if needed:

```bash
sudo mount /dev/nvme0n1p3 /mnt
sudo vi /mnt/etc/fstab
# Update UUIDs if changed
```

### GParted shows partition as "busy"

```bash
# In live USB terminal
sudo swapoff -a
sudo umount /dev/nvme0n1p1
sudo umount /dev/nvme0n1p3
```

### Resize operation fails

- Ensure no partitions are mounted
- Check disk for errors: `sudo fsck /dev/nvme0n1p1`
- Try from command line with `parted` if GParted fails

### Boot partition issues

The EFI partition (`/dev/nvme0n1p2`) should NOT be touched. If boot fails:

1. Boot from USB
2. Reinstall GRUB:

```bash
sudo mount /dev/nvme0n1p3 /mnt
sudo mount /dev/nvme0n1p2 /mnt/boot/efi
sudo grub-install --target=x86_64-efi --efi-directory=/mnt/boot/efi --boot-directory=/mnt/boot
sudo grub-mkconfig -o /mnt/boot/grub/grub.cfg
```

---

## Alternative: Symlink Strategy (No Repartition)

If you want to avoid repartitioning, keep moving large directories to /home:

| Directory | Size | Move Command |
|-----------|------|--------------|
| /var/lib/docker | ~12GB | See CRUSH-INSTALL.md |
| /var/lib/flatpak | ~4GB | `sudo mv /var/lib/flatpak /home/flatpak && sudo ln -s /home/flatpak /var/lib/flatpak` |
| /var/lib/snapd | ~3GB | Similar approach |
| ~/.ollama | ~18GB | Already moved via OLLAMA_MODELS |

This is safer but requires ongoing management.

---

## References

- [GParted Documentation](https://gparted.org/documentation.php)
- [Ubuntu Partition Guide](https://help.ubuntu.com/community/HowtoPartition)
- [Arch Wiki: Resizing Partitions](https://wiki.archlinux.org/title/Parted#Resizing_partitions)
