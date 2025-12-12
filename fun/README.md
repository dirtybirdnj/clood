# fun/

Silly tools and creative experiments. Because not everything has to be serious.

## bonsai.py

Generate colorful ASCII bonsai trees as SVGs using `cbonsai`.

### Requirements

```bash
# macOS
brew install cbonsai

# Linux
# Build from source: https://gitlab.com/jallbrit/cbonsai
```

### Usage

```bash
# Random tree to stdout
./bonsai.py

# Save to file
./bonsai.py -o my-tree.svg

# Big bushy tree
./bonsai.py --life 100 --multiplier 10 -o big-tree.svg

# Reproducible tree (same seed = same tree)
./bonsai.py --seed 42 -o answer-tree.svg

# Cherry blossom style
./bonsai.py --scheme cherry --leaf flowers -o sakura.svg

# Add a message
./bonsai.py --message "inner peace" -o zen.svg

# List all color schemes
./bonsai.py --list-colors

# List all leaf presets
./bonsai.py --list-leaves
```

### Color Schemes

| Scheme | Description |
|--------|-------------|
| `default` | Classic green/brown |
| `autumn` | Orange/brown fall colors |
| `cherry` | Pink cherry blossom |
| `winter` | White/silver |
| `neon` | Bright cyberpunk |
| `zen` | Muted, peaceful |
| `fire` | Red/orange flames |

### Leaf Presets

| Preset | Characters |
|--------|------------|
| `default` | `&` |
| `stars` | `*,✦,✧` |
| `hearts` | `♥,♡` |
| `flowers` | `✿,❀,✾` |
| `dots` | `●,○,◉` |
| `ascii` | `&,@,#` |
| `kanji` | `木,林,森` |
| `minimal` | `.` |

### Agent Usage

Agents can generate bonsai trees with specific parameters:

```bash
# Generate a unique tree for a session
./bonsai.py --seed $SESSION_ID --scheme zen -o session-tree.svg

# Generate themed trees
./bonsai.py --scheme cherry --leaf flowers --message "spring" -o spring.svg
./bonsai.py --scheme autumn --leaf default --message "fall" -o fall.svg
./bonsai.py --scheme winter --leaf minimal --message "winter" -o winter.svg
```

### Example Output

```
       &&
      &/|/&&
     &&~  &&&
    &&&\&&&
      &&|/
    &&&~~~
   &/\|/~_/
  /~//~
:___________./~~~\.___________:
 \                           /
  \_________________________/
  (_)                     (_)
```

(But with colors! The SVG preserves the ANSI colors from cbonsai.)
