// Package bonsai provides SVG generation for bonsai trees using Hershey single-line fonts.
package bonsai

import (
	"bytes"
	"embed"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//go:embed fonts/*.svg
var fontsFS embed.FS

// Glyph represents a single character's stroke path
type Glyph struct {
	Unicode  string
	Width    float64 // Horizontal advance
	PathData string  // SVG path d attribute
}

// HersheyFont contains parsed glyph data from a Hershey SVG font
type HersheyFont struct {
	Name       string
	UnitsPerEm float64
	Ascent     float64
	Descent    float64
	Glyphs     map[rune]Glyph
}

// SVGGenerator generates SVG from colored ASCII art
type SVGGenerator struct {
	Font       *HersheyFont
	CharWidth  float64 // Width per character cell
	CharHeight float64 // Height per character cell (line height)
	StrokeWidth float64
}

// LoadHersheyFont loads a Hershey font from the embedded fonts directory
func LoadHersheyFont(name string) (*HersheyFont, error) {
	filename := fmt.Sprintf("fonts/%s.svg", name)
	data, err := fontsFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("font not found: %s", name)
	}
	return parseHersheyFont(bytes.NewReader(data), name)
}

// parseHersheyFont parses an SVG font file into a HersheyFont structure
func parseHersheyFont(r io.Reader, name string) (*HersheyFont, error) {
	font := &HersheyFont{
		Name:       name,
		UnitsPerEm: 1000,
		Ascent:     800,
		Descent:    -200,
		Glyphs:     make(map[rune]Glyph),
	}

	decoder := xml.NewDecoder(r)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			switch elem.Name.Local {
			case "font-face":
				for _, attr := range elem.Attr {
					switch attr.Name.Local {
					case "units-per-em":
						font.UnitsPerEm, _ = strconv.ParseFloat(attr.Value, 64)
					case "ascent":
						font.Ascent, _ = strconv.ParseFloat(attr.Value, 64)
					case "descent":
						font.Descent, _ = strconv.ParseFloat(attr.Value, 64)
					}
				}
			case "glyph":
				var unicode string
				var width float64 = 378 // Default width
				var pathData string

				for _, attr := range elem.Attr {
					switch attr.Name.Local {
					case "unicode":
						unicode = attr.Value
					case "horiz-adv-x":
						width, _ = strconv.ParseFloat(attr.Value, 64)
					case "d":
						pathData = attr.Value
					}
				}

				if unicode != "" && len([]rune(unicode)) == 1 {
					r := []rune(unicode)[0]
					font.Glyphs[r] = Glyph{
						Unicode:  unicode,
						Width:    width,
						PathData: pathData,
					}
				}
			}
		}
	}

	return font, nil
}

// NewSVGGenerator creates a new SVG generator with the specified font
func NewSVGGenerator(fontName string) (*SVGGenerator, error) {
	font, err := LoadHersheyFont(fontName)
	if err != nil {
		return nil, err
	}

	// Calculate character dimensions based on font metrics
	// Normalize to a reasonable cell size for ASCII art
	scale := 10.0 / font.UnitsPerEm * 1000 // Normalize to ~10 units per character
	charWidth := 378.0 * scale / 1000      // Default glyph width normalized
	charHeight := (font.Ascent - font.Descent) * scale / 1000

	return &SVGGenerator{
		Font:        font,
		CharWidth:   charWidth,
		CharHeight:  charHeight,
		StrokeWidth: 0.3, // Thin stroke for pen plotter
	}, nil
}

// GenerateSVG creates an SVG from a BonsaiResult
func (g *SVGGenerator) GenerateSVG(result BonsaiResult) string {
	var buf bytes.Buffer

	// Calculate SVG dimensions
	width := float64(result.Width) * g.CharWidth
	height := float64(result.Height) * g.CharHeight

	// Add some margin
	margin := g.CharWidth * 2
	totalWidth := width + margin*2
	totalHeight := height + margin*2

	// SVG header
	buf.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg"
     width="%.2f" height="%.2f"
     viewBox="0 0 %.2f %.2f">
`, totalWidth, totalHeight, totalWidth, totalHeight))

	// Style for single-line strokes
	buf.WriteString(fmt.Sprintf(`<style>
  path { fill: none; stroke-width: %.2f; stroke-linecap: round; stroke-linejoin: round; }
</style>
`, g.StrokeWidth))

	// Sort layers by color for consistent output
	layers := result.Layers
	sort.Slice(layers, func(i, j int) bool {
		return layers[i].Color < layers[j].Color
	})

	// Generate a group for each color layer
	for _, layer := range layers {
		if len(layer.Characters) == 0 {
			continue
		}

		hexColor := ANSIColorToHex(layer.Color)
		layerName := layer.ColorName
		if layerName == "" {
			layerName = "default"
		}

		buf.WriteString(fmt.Sprintf(`<g id="layer-%s" stroke="%s">
`, layerName, hexColor))

		// Generate path for each character
		for _, char := range layer.Characters {
			path := g.getCharPath(char.Char, float64(char.Col)*g.CharWidth+margin, float64(char.Row)*g.CharHeight+margin)
			if path != "" {
				buf.WriteString(fmt.Sprintf(`  <path d="%s"/>
`, path))
			}
		}

		buf.WriteString("</g>\n")
	}

	buf.WriteString("</svg>\n")
	return buf.String()
}

// getCharPath generates an SVG path for a character at the specified position
func (g *SVGGenerator) getCharPath(char rune, x, y float64) string {
	glyph, ok := g.Font.Glyphs[char]
	if !ok || glyph.PathData == "" {
		return ""
	}

	// Transform the glyph path to the correct position and scale
	// Hershey fonts use Y-up coordinate system, SVG uses Y-down
	// Scale and translate the path
	scale := g.CharWidth / glyph.Width * 0.9 // Slight reduction for spacing
	return transformPath(glyph.PathData, x, y+g.CharHeight*0.8, scale, -scale)
}

// transformPath applies translation and scaling to an SVG path
func transformPath(pathData string, tx, ty, sx, sy float64) string {
	// Parse path commands and transform coordinates
	// Hershey fonts use simple M and L commands
	re := regexp.MustCompile(`([ML])\s*(-?[\d.]+)[,\s]+(-?[\d.]+)`)

	var result strings.Builder
	lastEnd := 0

	matches := re.FindAllStringSubmatchIndex(pathData, -1)
	for _, match := range matches {
		// Write any text between matches
		if match[0] > lastEnd {
			result.WriteString(pathData[lastEnd:match[0]])
		}

		cmd := pathData[match[2]:match[3]]
		xStr := pathData[match[4]:match[5]]
		yStr := pathData[match[6]:match[7]]

		x, _ := strconv.ParseFloat(xStr, 64)
		y, _ := strconv.ParseFloat(yStr, 64)

		// Transform coordinates
		newX := x*sx + tx
		newY := y*sy + ty

		result.WriteString(fmt.Sprintf("%s%.2f,%.2f", cmd, newX, newY))
		lastEnd = match[1]
	}

	// Write remaining text
	if lastEnd < len(pathData) {
		result.WriteString(pathData[lastEnd:])
	}

	return result.String()
}

// AvailableFonts returns the list of available Hershey fonts
func AvailableFonts() []string {
	entries, err := fontsFS.ReadDir("fonts")
	if err != nil {
		return nil
	}

	var fonts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".svg") {
			fonts = append(fonts, strings.TrimSuffix(entry.Name(), ".svg"))
		}
	}
	return fonts
}
