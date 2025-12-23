# Bonsai: ASCII-to-SVG for Pen Plotters

This package converts cbonsai ASCII art output into clean SVG suitable for pen plotters.

## Features

- **ANSI Parser**: Extracts clean ASCII from ncurses/terminal output
- **Color Extraction**: Separates characters by ANSI color into distinct layers
- **Single-Line Font Rendering**: Converts characters to SVG paths using Hershey-style stroke fonts

## Usage

### CLI

```bash
# Default: clean ASCII output
clood bonsai

# ASCII with size options
clood bonsai --size tiny
clood bonsai --size ancient

# SVG output with -o flag
clood bonsai --format svg -o bonsai.svg

# SVG piped to file (equivalent to -o)
clood bonsai -f svg > bonsai.svg

# SVG with custom font
clood bonsai -f svg --font EMSDelight -o bonsai-delight.svg
clood bonsai -f svg --font EMSCasualHand > casual.svg

# Reproducible output with seed
clood bonsai -f svg --seed 42 --size large -o bonsai.svg

# Pipe to other tools
clood bonsai -f svg --seed 42 | rat-king fill -p lines -o filled.svg
```

### Available Fonts

- `HersheySans1` - Classic single-stroke sans-serif (default)
- `EMSDelight` - Elegant cursive script
- `EMSCasualHand` - Casual handwritten style

### Programmatic Use

```go
import "github.com/dirtybirdnj/clood/internal/bonsai"

// Parse cbonsai output
rawOutput := runCbonsai()
result := bonsai.ParseANSIWithColors(rawOutput)

// result.ASCII - clean ASCII text
// result.Layers - characters grouped by color
// result.Width, result.Height - dimensions

// Generate SVG
gen, _ := bonsai.NewSVGGenerator("EMSDelight")
svg := gen.GenerateSVG(result)
```

## Output Structure

The SVG output contains:

1. **Color-separated layers** as `<g id="layer-{color}">` elements:
   - `layer-green` - Main foliage (ANSI 32)
   - `layer-bright-green` - Bright foliage (ANSI 92)
   - `layer-yellow` - Autumn colors (ANSI 33)
   - `layer-bright-yellow` - Bright accents (ANSI 93)
   - `layer-bright-black` - Pot/base (ANSI 90)

2. **Single-line paths** using Hershey/EMS stroke fonts - no filled shapes, just strokes suitable for pen plotters.

## Integration with rat-king

The output SVG can be processed by rat-king for additional pen plotter operations:

```bash
# Generate bonsai SVG
clood bonsai --format svg --seed 42 -o bonsai.svg

# Use rat-king to add pattern fills or process further
rat-king fill bonsai.svg -p lines -o bonsai-filled.svg
```

### Planned rat-king Integration

A dedicated `rat-king bonsai` or `rat-king ascii` command could:

1. Accept ASCII input or run cbonsai directly
2. Parse colors and positions
3. Convert to single-line font paths
4. Apply rat-king's line chaining and ordering
5. Output optimized SVG for plotting

The existing rat-king infrastructure provides:
- Line chaining and optimization
- Polygon detection from paths
- Pattern fill generation
- Hand-drawn/sketchy effects

## Technical Details

### ANSI Parsing

cbonsai uses ncurses which outputs:
- Cursor positioning (`ESC[row;colH`)
- Color codes (`ESC[32m` for green)
- Alternate screen buffer (`ESC[?1049h/l`)

The parser builds a virtual 80x24 screen, tracks cursor position and color, and extracts the tree before the exit clear.

### Font Format

Hershey/EMS fonts are SVG fonts where each glyph is defined as stroke paths:

```xml
<glyph unicode="A" horiz-adv-x="567"
       d="M 378 662 L 126 0 M 378 662 L 630 0 M 220 220 L 536 220"/>
```

The paths use only `M` (move) and `L` (line) commands - perfect for plotters.

### Coordinate System

- Font coordinates are normalized to ~10 units per character
- Y is flipped (SVG Y-down vs font Y-up)
- Characters are spaced based on `horiz-adv-x` glyph width
