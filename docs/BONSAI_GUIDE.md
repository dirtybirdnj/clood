# Bonsai Tree Generation Guide

Generate ASCII bonsai trees for the Server Garden aesthetic.

## Prerequisites

```bash
# Install cbonsai (macOS)
brew install cbonsai

# Install ansisvg for SVG export (Go required)
go install github.com/wader/ansisvg@latest
# Or download binary from: https://github.com/wader/ansisvg/releases
```

## Quick Start

```bash
# Generate a static tree (no animation)
cbonsai -p

# Generate and save as SVG
cbonsai -p | ansisvg > tree.svg
```

## Tree Size Control

The `-L` (life) and `-M` (multiplier) flags control tree shape:

| Size | Command | Description |
|------|---------|-------------|
| Tiny | `cbonsai -p -L 10 -M 2` | Small seedling |
| Small | `cbonsai -p -L 20 -M 3` | Young tree |
| Medium | `cbonsai -p -L 32 -M 5` | Default |
| Large | `cbonsai -p -L 60 -M 8` | Mature tree |
| Ancient | `cbonsai -p -L 100 -M 12` | Old growth |
| Maximum | `cbonsai -p -L 200 -M 20` | Absolute maximum |

## Tree Shape Styles

### Tall and Spindly
```bash
cbonsai -p -L 80 -M 3
```
High life, low multiplier = tall with few branches

### Short and Bushy
```bash
cbonsai -p -L 25 -M 15
```
Low life, high multiplier = short with many branches

### Balanced Classic
```bash
cbonsai -p -L 50 -M 8
```
Middle values for traditional bonsai look

### Weeping Style
```bash
cbonsai -p -L 60 -M 6 -c "~,_,."
```
Custom leaf characters for drooping effect

## Base/Pot Options

```bash
cbonsai -p -b 0   # No pot
cbonsai -p -b 1   # Small pot (default)
cbonsai -p -b 2   # Large pot
```

## Adding Messages

```bash
cbonsai -p -m "The Server Garden"
cbonsai -p -m "ubuntu25: ONLINE"
```

## SVG Export Options

```bash
# Basic SVG
cbonsai -p | ansisvg > tree.svg

# With custom font
cbonsai -p | ansisvg --fontname "JetBrains Mono" > tree.svg

# With specific width
cbonsai -p | ansisvg --width 80 > tree.svg

# Transparent background
cbonsai -p | ansisvg --transparent > tree.svg

# Different color schemes
ansisvg --listcolorschemes  # See available themes
cbonsai -p | ansisvg --colorscheme dracula > tree.svg
```

## Recipe: Generate Multiple Trees

```bash
# Generate a forest of different sizes
for size in 20 40 60 80; do
  cbonsai -p -L $size -M 5 | ansisvg > tree_${size}.svg
done

# Generate shape variations
cbonsai -p -L 80 -M 3 | ansisvg > tall.svg
cbonsai -p -L 25 -M 15 | ansisvg > bushy.svg
cbonsai -p -L 50 -M 8 | ansisvg > balanced.svg
```

## Interesting Variations

```bash
# Dense foliage
cbonsai -p -L 40 -M 20 -c "&"

# Cherry blossom style
cbonsai -p -L 50 -M 10 -c "*,o,."

# Winter/bare branches
cbonsai -p -L 60 -M 4 -c "."

# ASCII art leaves
cbonsai -p -L 45 -M 7 -c "{,}"
```

## Modular Output Suggestions

### For Different Hosts

```bash
# ubuntu25 - the Iron Keep (large, powerful)
cbonsai -p -L 80 -M 10 -m "ubuntu25" | ansisvg > ubuntu25.svg

# mac-mini - the Sentinel (compact, always-on)
cbonsai -p -L 35 -M 8 -m "mac-mini" | ansisvg > mac-mini.svg

# MacBook Air - the Jade Palace (elegant, portable)
cbonsai -p -L 50 -M 6 -c "~" -m "jade-palace" | ansisvg > macbook.svg
```

### For Status Indicators

```bash
# Healthy server
cbonsai -p -L 60 -M 8 -c "&,*" | ansisvg --colorscheme github > healthy.svg

# Degraded server
cbonsai -p -L 30 -M 4 | ansisvg --colorscheme solarized-dark > degraded.svg
```

## Troubleshooting

### Animation keeps playing
Always use `-p` flag for static output:
```bash
cbonsai -p  # Correct - prints final tree
cbonsai     # Wrong - will animate
```

### SVG has wrong colors
Try different color schemes or use `--transparent`:
```bash
cbonsai -p | ansisvg --colorscheme dracula --transparent > tree.svg
```

### Tree too wide for terminal
Reduce multiplier:
```bash
cbonsai -p -L 60 -M 4  # Narrower tree
```

## Integration Ideas

1. **clood banner**: Show small tree in startup banner
2. **clood garden**: Show tree for each host
3. **clood handoff**: Include tree as session "signature"
4. **Documentation**: Use SVG trees in README and docs

---

*"Tend your garden, tend your servers."*
