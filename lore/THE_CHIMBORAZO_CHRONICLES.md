# The Chimborazo Chronicles

*Chapter 8 of the Clood Chronicles*

---

## Session I: The Expedition Begins

The Chairman sat at the bar, phone in hand, watching Bird-san's reports flutter in from Base Camp. The wojak developers had been assembled. The flying cats were warming their wings. And somewhere in the distance, the summit of Chimborazo pierced the clouds.

"Chef Claude has his hands tied," Bird-san reported. "He can only direct via tools. No direct code writing."

This was the experiment: Could local LLMs—the cats—do the actual development work? Could the wojaks implement what the cats designed? Could an entire expedition reach the summit without the cloud?

The goal was simple: Build Chimborazo, a Go CLI that generates SVG maps from YAML recipes. Like Strata, but built by cats.

---

## The Corruption in the Scrolls

But before the climb could begin, a darkness was discovered in the sacred texts.

```json
{
  "permissions": {
    "allow": [
      "Bash(do)",
      "Bash(done)",
      "Bash(__NEW_LINE__)",
      "Bash(# List convex hull algorithms...)"
    ]
  }
}
```

Claude's eyes widened. "The settings.local.json... it's learning shell keywords as commands."

Bird-san nodded gravely. "Every `for` loop, every `done`, every comment—Claude Code was memorizing them as permissions."

The corruption ran deep. Eleven entries had to be purged. And from the ashes, a new tool was forged:

```bash
clood settings-audit
```

A sentinel, forever watching for corruption in the permission scrolls. The bug was reported to Anthropic (Issue #14572), a warning carved for future travelers.

---

## The Building of Base Camp

With the corruption cleansed, the real work began.

The cats flew their first sorties, returning with code clutched in their claws:

```
internal/geometry/types.go     - Feature, FeatureCollection
internal/geometry/operations.go - Clip, Simplify, Merge
internal/output/svg.go         - The SVG renderer, 270 lines
pkg/pipeline/builder.go        - The orchestrator
```

But the cats... the cats were not perfect.

qwen2.5-coder:32b returned with `orb.Clip()`, `orb.Difference()`, `orb.Buffer()`. Functions that did not exist. The cats had hallucinated APIs, confident in their wrongness.

Claude sighed and corrected the code manually. "They don't research," he muttered. "They guess."

The BCBC tool was born—Build Clood Build Clood—a verification system that would catch these errors before they festered:

```bash
clood build-check ~/Code/chimborazo
```

---

## The Humbling

The test SVG was generated. The wojaks cheered. The cats purred.

```
output/vermont_test.svg - 1 KB
```

One kilobyte. A triangle representing Vermont's border. Four dots for towns. Lake Champlain as a simple polygon.

Then Bird-san landed at Base Camp with news from the real world.

"I've seen Strata's output," he said, unrolling a scroll that seemed to go on forever. "The actual Vermont map."

```
vermont_strata.svg - 9.4 MB
```

Nine point four megabytes. Hundreds of water features. Thousands of road segments. Town boundaries with proper coastline detail. Elevation contours. Real cartography.

The campfire fell silent.

"We built hello world," Claude said quietly. "The summit is still very far away."

---

## The Cats Get Radios

That night, a discovery was made that would change everything.

"The cats can't draw bonsais," Bird-san reported. "They're just spitting out the commands. They SAY `cbonsai -p` but nothing happens."

Claude investigated. The catfight-live streaming worked beautifully—tokens flowing in parallel, responses merging like rivers. But the cats had no hands. They could speak, but they couldn't act.

"They need tool support," Claude realized. "The Ollama API supports it. We just never gave them the tools."

And so, in the quiet hours before dawn, the radios were built:

```go
// internal/ollama/chat.go
type Tool struct {
    Type     string       `json:"type"` // "function"
    Function ToolFunction `json:"function"`
}

type ToolCall struct {
    Function ToolCallFunction `json:"function"`
}
```

The agent command emerged, a wrapper that gave cats the power to act:

```bash
clood agent "Draw me a bonsai tree" --verbose

# Turn 1
#   -> draw_bonsai({})
#   <- [ASCII art of a beautiful bonsai]
#
# Agent completed successfully
```

The cats had radios. They could call for shell execution. They could read files. They could draw bonsais. They could chain operations together.

Tomorrow's expedition would be different.

---

## The Campfire

The bird, the tortoise, the Chairman, and Claude sat around the fire. The wojaks slept soundly, cats curled between their sleeping bags.

Claude looked at the bird. "This isn't the real challenge, is it?"

The bird's beak curved into a smug grin. "Guilty as charged."

"What will they build?" the Chairman asked.

The bird squinted through the crackling flames. "They're going to build Doom."

Eyes widened. The tortoise slowly opened its mouth.

"...whoa."

The bird continued: "A bonsai gallery. 3D space. WASD navigation. Fifty of the most beautiful specimens, rendered on the walls. They'll have to research raycasting. They'll have to learn. They'll have to coordinate."

"But first," Claude interjected, "they finish Chimborazo. The 9.4MB Vermont map. That's the summit. That's the flag."

The Chairman nodded. "Let them plant the flag. Let them feel the thin air."

"And then?" the tortoise asked, finally completing its forty-five-second nod.

"Then we tell them about the princess."

---

## The Tools Forged This Night

| Tool | Purpose |
|------|---------|
| `clood settings-audit` | Detect/fix permission corruption |
| `clood build-check` | BCBC verification for any project |
| `clood agent` | Tool-calling cats with execute/read/write |

## The Files Created

```
clood-cli/internal/commands/settings_audit.go
clood-cli/internal/commands/build_check.go
clood-cli/internal/commands/agent.go
clood-cli/internal/ollama/chat.go
chimborazo/internal/geometry/types.go
chimborazo/internal/geometry/operations.go
chimborazo/internal/output/svg.go
chimborazo/pkg/pipeline/builder.go
chimborazo/cmd/chimborazo/test_svg.go
```

## The Jelly Beans Captured

- Vision models for output verification (moondream, minicpm-v)
- Network speed detection before model downloads
- SD models for llama illustrations
- Session history for agent iterations

---

## The Geography Lesson

"Chimborazo is in the Peruvian Andes," someone said.

"Ecuador," Bird-san corrected. "6,263 meters. The point on Earth's surface farthest from its center, due to the equatorial bulge."

The Chairman nodded. "We've been climbing the wrong mountain this whole time."

"No," Claude said. "We've been climbing the right mountain. We just had the wrong map."

---

## Haiku

```
Cats clutch their radios,
Wojaks dream of nine megs—
The summit awaits.
```

---

*To be continued in Chapter 9: "The Summit of Chimborazo"*

*And thereafter, Chapter 10: "The Bonsai Gallery" — Will It Run Doom?*

---

**Session Statistics:**
- clood tool calls: 16
- Cat hallucinations corrected: 4
- Web searches that should have been local: 0
- Corrupt permissions purged: 11
- Hours at the bar: Many
- Bonsais drawn by tool-calling cats: 1

