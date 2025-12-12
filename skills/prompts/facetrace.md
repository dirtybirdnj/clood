# Facetrace - Bitmap to Vector Experimentation

## Background

dirtybirdnj has built a series of bitmap-to-vector tools for converting images to SVG paths, ultimately for plotter output (robotdrawsyou.com). The evolution:

1. **facetrace** (https://github.com/dirtybirdnj/facetrace) - Original web tool, jQuery + Caman.js + Potrace.js
2. **facetrace-react** (https://github.com/dirtybirdnj/facetrace-react) - React rewrite with layer support
3. **gellyscope** (https://github.com/dirtybirdnj/gellyscope) - Electron desktop app, most complete

## Current Tech Stack (gellyscope)

- **Electron** - Desktop framework
- **Potrace.js** - JavaScript port of potrace for bitmap-to-vector
- **Vpype** - Vector pipeline for G-code generation
- **Vanilla JS** - 15 modular components

## Potrace Parameters

| Parameter | Type | Purpose |
|-----------|------|---------|
| `turnpolicy` | enum (black, white, left, right, minority, majority) | Path decomposition strategy |
| `turdsize` | integer | Speckle suppression - minimum area to trace |
| `alphamax` | float | Corner detection threshold |
| `opttolerance` | float | Curve optimization tolerance |
| `optcurve` | boolean | Enable/disable curve optimization |

## Preprocessing Pipeline (gellyscope)

1. Image upload from ~/gellyscope directory
2. CSS filters: brightness, contrast, saturation, hue, grayscale, sepia
3. Image scaling (100%, 75%, 50%, 25%)
4. Sobel edge detection (optional)
5. Potrace conversion
6. Layer capture and stacking
7. SVG-to-G-code via vpype

## The Problem

No automated way to experiment with parameter combinations. Currently requires manual slider tweaking to find optimal settings for different image types.

## Proposed Solution: Parameter Sweep Tool

A CLI tool or gellyscope extension that:

1. Takes an input image + parameter ranges
2. Generates all combinations (grid search)
3. Outputs SVGs with parameter-encoded filenames
4. Creates HTML comparison gallery
5. Allows saving winning combos as presets

### Example Usage

```bash
node sweep.js input.png \
  --turdsize=2,5,10,20,50 \
  --alphamax=0.5,1.0,1.5 \
  --turnpolicy=minority,majority
```

Generates 30 variants (5 × 3 × 2) in `output/` with comparison HTML.

### Directory Structure

```
clood/skills/vectorization/
├── sweep.js          # CLI parameter sweep tool
├── compare.html      # Generated comparison gallery
├── presets/          # Saved parameter combinations
│   ├── faces.json
│   ├── line-art.json
│   └── high-contrast.json
└── README.md
```

## Future: ML-Assisted Parameter Selection

Eventually train a model to predict optimal parameters based on image characteristics:

- Input: image histogram, edge density, contrast ratio
- Output: recommended potrace settings
- Training data: user selections from sweep comparisons

## Key Files Reference

- **gellyscope:** `/src/trace.js` (1,280 lines - main tracing logic)
- **gellyscope:** `/potrace.js` (wrapper)
- **gellyscope:** `/main.js` (IPC handlers, file operations)
- **facetrace:** `/js/potrace.js` (parameter definitions)

## Related Projects

- **svg-grouper** - Local repo at /home/mgilbert/Code/svg-grouper
- **robotdrawsyou.com** - Production site using these tools
