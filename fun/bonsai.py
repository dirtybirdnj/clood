#!/usr/bin/env python3
"""
bonsai.py - Generate colorful ASCII bonsai trees as SVGs

Uses cbonsai to generate trees, captures ANSI output, converts to SVG.
Designed for agent use - all parameters controllable via CLI.

Usage:
    ./bonsai.py                           # Random tree to stdout
    ./bonsai.py -o tree.svg               # Save to file
    ./bonsai.py --life 100 --multiplier 8 # Bigger, bushier tree
    ./bonsai.py --seed 42 --message "zen" # Reproducible with message
    ./bonsai.py --list-colors             # Show available color schemes
"""

import argparse
import subprocess
import sys
import re
from pathlib import Path

# ANSI 256-color to hex mapping (subset of most common)
ANSI_TO_HEX = {
    0: "#000000", 1: "#800000", 2: "#008000", 3: "#808000",
    4: "#000080", 5: "#800080", 6: "#008080", 7: "#c0c0c0",
    8: "#808080", 9: "#ff0000", 10: "#00ff00", 11: "#ffff00",
    12: "#0000ff", 13: "#ff00ff", 14: "#00ffff", 15: "#ffffff",
    # Extended colors (greens, browns for trees)
    22: "#005f00", 28: "#008700", 34: "#00af00", 40: "#00d700",
    46: "#00ff00", 58: "#5f5f00", 64: "#5f8700", 70: "#5faf00",
    94: "#875f00", 100: "#878700", 106: "#87af00", 130: "#af5f00",
    136: "#af8700", 142: "#afaf00", 166: "#d75f00", 172: "#d78700",
    178: "#d7af00", 184: "#d7d700", 190: "#d7ff00", 208: "#ff8700",
    214: "#ffaf00", 220: "#ffd700", 226: "#ffff00",
}

# Preset color schemes
COLOR_SCHEMES = {
    "default": "2,3,10,11",      # Classic green/brown
    "autumn": "208,94,214,130",   # Orange/brown fall colors
    "cherry": "198,52,205,89",    # Pink cherry blossom
    "winter": "255,240,15,250",   # White/silver
    "neon": "46,201,226,196",     # Bright cyberpunk
    "zen": "22,58,28,94",         # Muted, peaceful
    "fire": "196,202,226,208",    # Red/orange flames
}

LEAF_PRESETS = {
    "default": "&",
    "stars": "*,✦,✧",
    "hearts": "♥,♡",
    "flowers": "✿,❀,✾",
    "dots": "●,○,◉",
    "ascii": "&,@,#",
    "kanji": "木,林,森",
    "minimal": ".",
}

def get_hex_color(ansi_code):
    """Convert ANSI color code to hex."""
    if ansi_code in ANSI_TO_HEX:
        return ANSI_TO_HEX[ansi_code]
    # Generate approximate hex for 256-color palette
    if 16 <= ansi_code <= 231:
        # Color cube: 6x6x6
        ansi_code -= 16
        b = ansi_code % 6
        g = (ansi_code // 6) % 6
        r = ansi_code // 36
        return f"#{r*51:02x}{g*51:02x}{b*51:02x}"
    if 232 <= ansi_code <= 255:
        # Grayscale
        gray = (ansi_code - 232) * 10 + 8
        return f"#{gray:02x}{gray:02x}{gray:02x}"
    return "#ffffff"

def strip_ansi_control(text):
    """Remove all ANSI control sequences except color codes."""
    # Remove cursor positioning, screen clearing, etc.
    text = re.sub(r'\x1b\[\?[0-9;]*[a-zA-Z]', '', text)  # Private sequences
    text = re.sub(r'\x1b\[[0-9;]*[HJKfABCDEFGsu]', '', text)  # Cursor/clear
    text = re.sub(r'\x1b\][^\x07]*\x07', '', text)  # OSC sequences
    text = re.sub(r'\x1b\([A-Z0-9]', '', text)  # Character set
    return text

def parse_ansi(text):
    """Parse ANSI text into list of (char, color) tuples."""
    # First strip non-color control sequences
    text = strip_ansi_control(text)

    result = []
    current_color = "#00ff00"  # Default green

    # Regex to match ANSI color escape sequences
    ansi_pattern = re.compile(r'\x1b\[([0-9;]*)m')

    i = 0
    while i < len(text):
        # Check for ANSI escape sequence
        match = ansi_pattern.match(text, i)
        if match:
            codes = match.group(1).split(';')
            for j, code in enumerate(codes):
                if code == '38' and j + 2 < len(codes) and codes[j + 1] == '5':
                    # 256-color foreground: \x1b[38;5;Nm
                    try:
                        current_color = get_hex_color(int(codes[j + 2]))
                    except (ValueError, IndexError):
                        pass
                elif code == '0':
                    current_color = "#00ff00"  # Reset
            i = match.end()
        elif text[i] == '\x1b':
            # Skip any remaining escape sequences
            i += 1
            while i < len(text) and text[i] not in 'mABCDEFGHJKSTfnsu':
                i += 1
            if i < len(text):
                i += 1
        else:
            char = text[i]
            if char.isprintable() or char in ' \t':
                result.append((char, current_color))
            i += 1

    return result

def ansi_to_svg(ansi_text, font_size=14, font_family="monospace", bg_color="#1a1a1a"):
    """Convert ANSI-colored text to SVG."""
    lines = ansi_text.split('\n')
    parsed_lines = [parse_ansi(line) for line in lines]

    # Calculate dimensions
    char_width = font_size * 0.6
    line_height = font_size * 1.2
    max_width = max(len(line) for line in parsed_lines) if parsed_lines else 0

    width = int(max_width * char_width + 40)
    height = int(len(parsed_lines) * line_height + 40)

    svg_parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}">',
        f'  <rect width="100%" height="100%" fill="{bg_color}"/>',
        f'  <text font-family="{font_family}" font-size="{font_size}px">'
    ]

    for line_idx, parsed_line in enumerate(parsed_lines):
        y = 20 + line_idx * line_height
        x = 20

        for char, color in parsed_line:
            if char == ' ':
                x += char_width
                continue
            # Escape XML special characters
            if char == '<':
                char = '&lt;'
            elif char == '>':
                char = '&gt;'
            elif char == '&':
                char = '&amp;'
            elif char == '"':
                char = '&quot;'

            svg_parts.append(
                f'    <tspan x="{x:.1f}" y="{y:.1f}" fill="{color}">{char}</tspan>'
            )
            x += char_width

    svg_parts.append('  </text>')
    svg_parts.append('</svg>')

    return '\n'.join(svg_parts)

def generate_bonsai(life=32, multiplier=5, seed=None, message=None,
                    base=1, leaf="&", colors="2,3,10,11"):
    """Generate bonsai ASCII art using cbonsai."""
    cmd = ["cbonsai", "-p"]  # -p = print (no animation)

    cmd.extend(["-L", str(life)])
    cmd.extend(["-M", str(multiplier)])
    cmd.extend(["-b", str(base)])
    cmd.extend(["-c", leaf])
    cmd.extend(["-k", colors])

    if seed is not None:
        cmd.extend(["-s", str(seed)])
    if message:
        cmd.extend(["-m", message])

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            env={**subprocess.os.environ, "TERM": "xterm-256color"}
        )
        return result.stdout
    except FileNotFoundError:
        print("Error: cbonsai not found. Install with: brew install cbonsai", file=sys.stderr)
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(
        description="Generate colorful ASCII bonsai trees as SVGs",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                              # Random tree, SVG to stdout
  %(prog)s -o tree.svg                  # Save to file
  %(prog)s --life 100 --multiplier 10   # Big bushy tree
  %(prog)s --seed 42                    # Reproducible tree
  %(prog)s --scheme cherry --leaf flowers  # Cherry blossom style
  %(prog)s --message "inner peace"      # Add a message

Color schemes: """ + ", ".join(COLOR_SCHEMES.keys()) + """
Leaf presets: """ + ", ".join(LEAF_PRESETS.keys())
    )

    parser.add_argument("-o", "--output", help="Output SVG file (default: stdout)")
    parser.add_argument("-L", "--life", type=int, default=32,
                        help="Tree life/size (0-200) [default: 32]")
    parser.add_argument("-M", "--multiplier", type=int, default=5,
                        help="Branch multiplier (0-20) [default: 5]")
    parser.add_argument("-s", "--seed", type=int, help="Random seed for reproducibility")
    parser.add_argument("-m", "--message", help="Message to display next to tree")
    parser.add_argument("-b", "--base", type=int, default=1,
                        help="Pot/base style (0=none, 1=default, 2=large)")
    parser.add_argument("--scheme", choices=COLOR_SCHEMES.keys(), default="default",
                        help="Color scheme preset")
    parser.add_argument("--colors", help="Custom colors: dark_leaf,dark_wood,light_leaf,light_wood")
    parser.add_argument("--leaf", default="default",
                        help="Leaf preset name or custom chars (comma-separated)")
    parser.add_argument("--font-size", type=int, default=14, help="SVG font size")
    parser.add_argument("--bg", default="#1a1a1a", help="Background color")
    parser.add_argument("--ascii", action="store_true", help="Output raw ASCII instead of SVG")
    parser.add_argument("--list-colors", action="store_true", help="List color schemes")
    parser.add_argument("--list-leaves", action="store_true", help="List leaf presets")

    args = parser.parse_args()

    if args.list_colors:
        print("Color schemes:")
        for name, colors in COLOR_SCHEMES.items():
            print(f"  {name}: {colors}")
        return

    if args.list_leaves:
        print("Leaf presets:")
        for name, chars in LEAF_PRESETS.items():
            print(f"  {name}: {chars}")
        return

    # Resolve colors
    colors = args.colors if args.colors else COLOR_SCHEMES[args.scheme]

    # Resolve leaf characters
    leaf = LEAF_PRESETS.get(args.leaf, args.leaf)

    # Generate the bonsai
    ansi_output = generate_bonsai(
        life=args.life,
        multiplier=args.multiplier,
        seed=args.seed,
        message=args.message,
        base=args.base,
        leaf=leaf,
        colors=colors
    )

    if args.ascii:
        output = ansi_output
    else:
        output = ansi_to_svg(
            ansi_output,
            font_size=args.font_size,
            bg_color=args.bg
        )

    if args.output:
        Path(args.output).write_text(output)
        print(f"Saved to {args.output}", file=sys.stderr)
    else:
        print(output)

if __name__ == "__main__":
    main()
