package sd

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// CompareGallery generates an HTML comparison view for batch results.
type CompareGallery struct {
	Title       string
	Description string
	Results     *BatchResult
	OutputPath  string
}

// NewCompareGallery creates a gallery from batch results.
func NewCompareGallery(results *BatchResult) *CompareGallery {
	return &CompareGallery{
		Title:       results.Config.Name,
		Description: results.Config.Description,
		Results:     results,
		OutputPath:  filepath.Join(results.Config.OutputDir, "compare.html"),
	}
}

// Render writes the HTML gallery to the given writer.
func (g *CompareGallery) Render(w io.Writer) error {
	tmpl, err := template.New("gallery").Parse(galleryTemplate)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Title       string
		Description string
		Prompt      string
		Negative    string
		Seed        int64
		Generated   string
		TotalTime   string
		Results     []VariationResult
	}{
		Title:       g.Title,
		Description: g.Description,
		Prompt:      g.Results.Config.BasePrompt.FormatPositive(),
		Negative:    g.Results.Config.BasePrompt.Negative,
		Seed:        g.Results.Config.BasePrompt.Seed,
		Generated:   g.Results.EndTime.Format(time.RFC3339),
		TotalTime:   g.Results.TotalTime.Round(time.Second).String(),
		Results:     g.Results.Results,
	}

	return tmpl.Execute(w, data)
}

const galleryTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Catfight Results</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #1a1a2e;
            color: #eee;
            padding: 2rem;
        }
        .header {
            max-width: 1400px;
            margin: 0 auto 2rem;
            padding-bottom: 1rem;
            border-bottom: 2px solid #e94560;
        }
        h1 {
            color: #e94560;
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
        }
        .subtitle { color: #888; font-style: italic; }
        .prompt-box {
            background: #16213e;
            border-radius: 8px;
            padding: 1rem;
            margin: 1rem 0;
            font-family: monospace;
            font-size: 0.9rem;
            border-left: 4px solid #e94560;
        }
        .prompt-box .label {
            color: #e94560;
            font-weight: bold;
            margin-bottom: 0.5rem;
        }
        .meta {
            display: flex;
            gap: 2rem;
            margin-top: 1rem;
            color: #888;
            font-size: 0.9rem;
        }
        .gallery {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 2rem;
            max-width: 1400px;
            margin: 0 auto;
        }
        .card {
            background: #16213e;
            border-radius: 12px;
            overflow: hidden;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 32px rgba(233, 69, 96, 0.3);
        }
        .card img {
            width: 100%;
            height: auto;
            display: block;
            cursor: pointer;
        }
        .card-body {
            padding: 1rem;
        }
        .card-title {
            color: #e94560;
            font-size: 1.2rem;
            margin-bottom: 0.5rem;
        }
        .card-meta {
            font-size: 0.85rem;
            color: #888;
        }
        .card-meta span {
            display: inline-block;
            background: #0f3460;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
            margin: 0.2rem 0.2rem 0.2rem 0;
        }
        .success { border-top: 3px solid #4ecca3; }
        .failure { border-top: 3px solid #ff6b6b; opacity: 0.7; }
        .error-msg {
            color: #ff6b6b;
            font-style: italic;
            padding: 1rem;
        }
        /* Lightbox */
        .lightbox {
            display: none;
            position: fixed;
            top: 0; left: 0;
            width: 100%; height: 100%;
            background: rgba(0,0,0,0.95);
            z-index: 1000;
            cursor: pointer;
        }
        .lightbox.active { display: flex; align-items: center; justify-content: center; }
        .lightbox img {
            max-width: 95%;
            max-height: 95%;
            object-fit: contain;
        }
        .footer {
            text-align: center;
            margin-top: 3rem;
            padding-top: 1rem;
            border-top: 1px solid #333;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Title}}</h1>
        {{if .Description}}<p class="subtitle">{{.Description}}</p>{{end}}

        <div class="prompt-box">
            <div class="label">Positive Prompt:</div>
            {{.Prompt}}
        </div>
        <div class="prompt-box">
            <div class="label">Negative Prompt:</div>
            {{.Negative}}
        </div>

        <div class="meta">
            <span>Seed: {{.Seed}}</span>
            <span>Generated: {{.Generated}}</span>
            <span>Total Time: {{.TotalTime}}</span>
        </div>
    </div>

    <div class="gallery">
        {{range .Results}}
        <div class="card {{if .Success}}success{{else}}failure{{end}}">
            {{if .Success}}
            <img src="{{.OutputPath}}" alt="{{.Variation.Name}}" onclick="openLightbox(this.src)">
            {{else}}
            <div class="error-msg">Generation failed: {{.Error}}</div>
            {{end}}
            <div class="card-body">
                <div class="card-title">{{.Variation.Name}}</div>
                <div class="card-meta">
                    <span>{{.Metadata.Checkpoint}}</span>
                    {{range .Metadata.LoRAs}}
                    <span>{{.Name}}: {{.Weight}}</span>
                    {{end}}
                    <span>{{.GenerateTime}}</span>
                </div>
            </div>
        </div>
        {{end}}
    </div>

    <div class="lightbox" id="lightbox" onclick="closeLightbox()">
        <img id="lightbox-img" src="" alt="Full size">
    </div>

    <div class="footer">
        Generated by clood sd catfight
    </div>

    <script>
        function openLightbox(src) {
            document.getElementById('lightbox-img').src = src;
            document.getElementById('lightbox').classList.add('active');
        }
        function closeLightbox() {
            document.getElementById('lightbox').classList.remove('active');
        }
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') closeLightbox();
        });
    </script>
</body>
</html>`

// MarkdownReport generates a markdown summary of the batch results.
func MarkdownReport(results *BatchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", results.Config.Name))
	if results.Config.Description != "" {
		sb.WriteString(fmt.Sprintf("*%s*\n\n", results.Config.Description))
	}

	sb.WriteString("## Prompt\n\n")
	sb.WriteString(fmt.Sprintf("**Positive:** %s\n\n", results.Config.BasePrompt.FormatPositive()))
	sb.WriteString(fmt.Sprintf("**Negative:** %s\n\n", results.Config.BasePrompt.Negative))
	sb.WriteString(fmt.Sprintf("**Seed:** %d\n\n", results.Config.BasePrompt.Seed))

	sb.WriteString("## Results\n\n")
	sb.WriteString("| Variation | Checkpoint | LoRA | Time | Status |\n")
	sb.WriteString("|-----------|------------|------|------|--------|\n")

	for _, r := range results.Results {
		status := "Success"
		if !r.Success {
			status = fmt.Sprintf("Failed: %s", r.Error)
		}

		loraStr := "-"
		if len(r.Metadata.LoRAs) > 0 {
			var parts []string
			for _, l := range r.Metadata.LoRAs {
				parts = append(parts, fmt.Sprintf("%s@%.2f", l.Name, l.Weight))
			}
			loraStr = strings.Join(parts, ", ")
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			r.Variation.Name,
			r.Metadata.Checkpoint,
			loraStr,
			r.GenerateTime.Round(time.Millisecond),
			status,
		))
	}

	sb.WriteString(fmt.Sprintf("\n**Total Time:** %s\n", results.TotalTime.Round(time.Second)))

	return sb.String()
}
