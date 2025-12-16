# Legend of Clood: Image Generation Recipes

A recipe book for generating visual assets using Stable Diffusion. No configuration required—this documents what to generate and how, for when the infrastructure is ready.

---

## The Vision

The Legend of Clood uses imagery to make AI infrastructure concepts accessible to different audiences:

| Audience | Style | Purpose |
|----------|-------|---------|
| Developers | Cyberpunk + Japanese | Technical cool factor |
| Non-technical | Studio Ghibli-like | Approachable, whimsical |
| Kids/Families | Cartoon/illustrated | Friendly, educational |
| Art lovers | Traditional ukiyo-e | Cultural authenticity |
| Marketing | Polished fantasy | Professional appeal |

---

## Part I: Bird-san Character Variations

Bird-san is a rock pigeon (Kawara-bato) ronin. Here are 10+ variations to test different aesthetics.

### 1. Traditional Ukiyo-e

**Model:** [Evo-Ukiyoe-v1](https://huggingface.co/SakanaAI/Evo-Ukiyoe-v1) or [ukiyo-e-art LoRA](https://huggingface.co/KappaNeuro/ukiyo-e-art)

```
A Japanese rock pigeon wearing a white hachimaki headband and
tattered gray kimono, standing proudly on a stone lantern.
Traditional ukiyo-e woodblock print style, flat colors, bold
outlines, Edo period aesthetic, muted earth tones with accent
of indigo blue.

Negative: modern, 3D, photorealistic, western
```

### 2. Sumi-e Ink Wash

**Model:** [sdxl_chinese_ink_lora](https://huggingface.co/ming-yang/sdxl_chinese_ink_lora) or Shukezouma/MoXin

```
sumieink wash painting of a small pigeon samurai, minimal
brushstrokes, black ink on rice paper, negative space,
traditional Chinese/Japanese ink painting, contemplative
pose, bamboo and mist in background.

Negative: color, detailed, busy, photorealistic
```

### 3. Studio Ghibli Style

**Model:** Ghibli-style LoRA or DreamShaper

```
A cute anthropomorphic pigeon character in the style of
Studio Ghibli, wearing a small kimono and headband, sitting
on a rooftop overlooking a misty valley with glowing servers
hidden among trees. Soft watercolor palette, warm lighting,
whimsical, detailed feathers, expressive eyes.

Negative: dark, scary, realistic, western animation
```

### 4. Cyberpunk Samurai

**Model:** Cyberpunk edge runners style or StarLight XL

```
A cyberpunk pigeon ronin, neon-lit Tokyo alley, glowing
circuit patterns on kimono, holographic hachimaki, one
cybernetic eye, rain-slicked streets, kanji neon signs,
server racks visible through windows. Blade Runner aesthetic,
high contrast, cinematic.

Negative: cute, cartoon, bright colors, daytime
```

### 5. Children's Book Illustration

**Model:** Cartoon/illustration style checkpoint

```
A friendly cartoon pigeon dressed as a little samurai,
simple shapes, bold colors, children's book illustration
style, wearing a tiny blue kimono with cloud patterns,
white headband, standing in a garden of computer flowers.
Cheerful, educational, approachable.

Negative: scary, detailed, realistic, violent
```

### 6. Realistic Fantasy

**Model:** DreamShaper or Animagine XL

```
A photorealistic rock pigeon (Columba livia) perched on
ancient server equipment, wearing a miniature samurai
headband. Fantasy art style, dramatic lighting, misty
mountain background, blend of nature and technology.
Detailed feathers, iridescent neck, dignified pose.

Negative: cartoon, anime, flat colors
```

### 7. Emakimono (Scroll Painting)

**Model:** ukiyo-e-art with landscape focus

```
A horizontal scroll painting depicting a pigeon's journey
across landscapes. Left: departing a golden palace. Center:
crossing misty mountains with server towers. Right: arriving
at a humble garden. Traditional emakimono format, narrative
flow, gold leaf accents.

Negative: single scene, portrait, modern
```

### 8. Manga/Comic Style

**Model:** Animagine XL or anime checkpoint

```
Dynamic manga-style illustration of Bird-san in action pose,
speed lines, dramatic angle from below, wind-swept kimono,
determined expression, black and white with screen tones,
Japanese manga aesthetic, bold linework.

Negative: color, western comic, soft, static
```

### 9. Edo Period Historical

**Model:** Evo-Ukiyoe-v1 with historical prompt

```
A historical ukiyo-e print depicting a noble pigeon in the
style of Hiroshige or Hokusai. Mount Fuji in background,
traditional composition, woodblock texture, period-accurate
colors (indigo, ochre, vermillion), nature scene with
subtle technology hints.

Negative: modern, digital, anime, futuristic
```

### 10. Watercolor Journal

**Model:** Realistic watercolor LoRA

```
A naturalist's journal entry: watercolor sketch of a rock
pigeon wearing a small cloth headband, field notes in
handwritten Japanese around the margins, botanical
illustration style, educational, scientific yet artistic.

Negative: digital, clean, finished, cartoon
```

### 11. Neon Noir

**Model:** Noir or cyberpunk checkpoint

```
Film noir style pigeon detective in a rain-soaked alley,
dramatic shadows, single spotlight, wearing a tiny fedora
over the hachimaki, server glow in background, cinematic
composition, black and white with subtle color accents.

Negative: bright, cheerful, cartoon
```

### 12. Kaiju Poster

**Model:** Fantasy/monster checkpoint

```
Movie poster style: giant Bird-san towering over a city of
server buildings, dramatic low angle, storm clouds,
lightning, Japanese monster movie aesthetic, bold title
space at bottom, retro 1960s tokusatsu feel.

Negative: cute, small, peaceful
```

---

## Part II: Recommended Models by Style

### Japanese Traditional

| Model | Source | Best For |
|-------|--------|----------|
| Evo-Ukiyoe-v1 | [HuggingFace](https://huggingface.co/SakanaAI/Evo-Ukiyoe-v1) | Authentic woodblock prints |
| ukiyo-e-art | [HuggingFace](https://huggingface.co/KappaNeuro/ukiyo-e-art) | General ukiyo-e style |
| sdxl_chinese_ink_lora | [HuggingFace](https://huggingface.co/ming-yang/sdxl_chinese_ink_lora) | Ink wash/sumi-e |
| MoXin/Shukezouma | Civitai | Chinese landscape painting |

### Fantasy & Stylized

| Model | Source | Best For |
|-------|--------|----------|
| DreamShaper | Civitai | Fantasy realism blend |
| StarLight XL | Civitai | Ethereal/mystical |
| Pony Diffusion V6 XL | [Open Laboratory](https://openlaboratory.ai/models/pony-diffusion-v6-xl) | Anthropomorphic characters |
| Animagine XL 4.0 | Civitai | Anime/manga style |

### Technical Style Terms

Add these to prompts for specific effects:

```
# For traditional Japanese
ukiyo-e, woodblock print, Edo period, flat colors, bold outlines

# For ink wash
sumi-e, ink wash, rice paper, negative space, brushstrokes

# For cyberpunk
neon, rain, high contrast, holographic, circuit patterns

# For Ghibli-like
Studio Ghibli style, soft watercolor, warm lighting, whimsical

# For fantasy
fantasy art, dramatic lighting, epic, detailed, painterly
```

---

## Part III: Scene Recipes

### The Server Garden

```
A Japanese zen garden where the rocks are server racks, the
raked sand has binary patterns, bonsai trees grow from
ethernet cables, mist rising from cooling vents, morning
light, peaceful yet technological. Studio Ghibli meets
cyberpunk.
```

### The Jade Palace (MacBook)

```
A gleaming jade tower in the clouds, translucent walls
showing flowing data streams, elegant and portable, a noble
fox spirit (kitsune) visible in the highest window. Chinese
fantasy architecture, luminous, ethereal.
```

### The Iron Keep (ubuntu25)

```
A dark fortress on a volcanic mountain, fires burning inside,
smoke from cooling systems, a fierce tengu (crow demon)
perched on the battlements, heavy stone walls with circuit
engravings. Dark fantasy, powerful, industrial.
```

### The Sentinel (mac-mini)

```
A small but vigilant stone guardian at a mountain pass,
glowing eyes always watching, covered in moss but humming
with energy, compact and reliable. Studio Ghibli kodama
meets technology.
```

### The Tanuki (Ollama)

```
A mischievous tanuki (raccoon dog) spirit transforming
between different forms—scholar, warrior, poet—each form
slightly different, surrounded by floating kanji characters.
Traditional Japanese yokai illustration with modern twist.
```

### The Gashadokuro (Cloud Giants)

```
A massive skeleton made of server racks and cables, looming
over a city, eating streams of data from the clouds,
terrifying but distant. Dark fantasy, ominous, scale
emphasized, inspired by Goya and Japanese yokai.
```

---

## Part IV: Batch Generation Strategy

### Phase 1: Character Exploration (10 images)

Generate Bird-san in all 10+ styles to find which resonates:

```bash
# When infrastructure is ready:
clood imagine "Bird-san ukiyo-e" --model evo-ukiyoe --save bird-san-001
clood imagine "Bird-san sumi-e" --model chinese-ink --save bird-san-002
clood imagine "Bird-san ghibli" --model dreamshaper --save bird-san-003
# ... etc
```

### Phase 2: Style Refinement (5 images per winner)

Take the top 3 styles and generate variations:

```bash
clood imagine "Bird-san ukiyo-e, morning light" --batch 5 --variations
clood imagine "Bird-san ukiyo-e, dramatic pose" --batch 5 --variations
```

### Phase 3: Scene Generation

Use winning Bird-san style for all narrative scenes.

### Phase 4: Demographic Variants

Generate same scene in multiple styles for different audiences:

```bash
# Technical audience
clood imagine "Server Garden cyberpunk" --style neon-noir

# General audience
clood imagine "Server Garden ghibli" --style warm-whimsical

# Kids
clood imagine "Server Garden cartoon" --style friendly-educational
```

---

## Part V: Prompt Engineering Patterns

### Base Pattern for Bird-san

```
[style modifier] [subject: rock pigeon/kawara-bato] [clothing: hachimaki, kimono]
[action/pose] [setting/background] [lighting] [technical style keywords]

Negative: [things to avoid for this style]
```

### Style Modifiers by Demographic

| Audience | Add to Prompt | Remove from Prompt |
|----------|---------------|-------------------|
| Developers | "cinematic, detailed, professional" | "cute, cartoon" |
| Non-tech | "friendly, approachable, warm" | "technical, dark" |
| Kids | "cheerful, simple, colorful" | "scary, complex" |
| Art lovers | "museum quality, authentic" | "anime, digital" |

### Quality Boosters

```
# For all styles
masterpiece, best quality, highly detailed

# For traditional
authentic, museum quality, traditional technique

# For fantasy
epic, dramatic, cinematic lighting

# For cute
adorable, charming, heartwarming
```

---

## Part VI: Model Download List

When ready to set up, download these:

### Must Have (Core Styles)

1. **SDXL Base** - Foundation
2. **Evo-Ukiyoe-v1** - Japanese woodblock
3. **DreamShaper** - Fantasy blend
4. **sdxl_chinese_ink_lora** - Ink wash

### Nice to Have (Variety)

5. **Animagine XL** - Anime/manga
6. **Pony Diffusion V6** - Anthropomorphic
7. **StarLight XL** - Ethereal fantasy
8. **Cyberpunk LoRA** - Neon aesthetic

### File Locations

```
~/.clood/models/
├── checkpoints/
│   ├── sdxl-base.safetensors
│   ├── dreamshaper.safetensors
│   └── ...
└── loras/
    ├── ukiyo-e-art.safetensors
    ├── chinese-ink.safetensors
    └── ...
```

---

## Part VII: The Sauce Metaphor

> *"The Legend of Clood is putting tasty sauce on vegetables to help explain different tiers of the AI system to different demographics or ages"*

### How Images Work as Sauce

| Technical Concept | Without Sauce | With Sauce |
|------------------|---------------|------------|
| Multi-machine routing | "Load balancing across Ollama instances" | "The Tanuki shapeshifts between keeps" |
| Context window limits | "Token limit exceeded" | "The Kappa's bowl runs dry" |
| Local vs cloud trade-off | "Latency vs cost optimization" | "Growing your garden vs begging at the Emperor's gate" |
| Background processing | "Async job queue" | "The garden grows while you sleep" |

### Image + Text = Understanding

Each image should:
1. **Embody** a technical concept visually
2. **Evoke** an emotional response
3. **Explain** without requiring technical knowledge
4. **Remember** - be memorable enough to recall the concept

---

## Appendix: Quick Reference Card

### Generate Bird-san

```bash
# Quick test
cbonsai -p | ansisvg > tree.svg  # ASCII placeholder until SD ready

# When SD ready
clood imagine "Bird-san [style]" --model [model] --save [name]
```

### Style Keywords Cheat Sheet

```
Traditional:  ukiyo-e, sumi-e, woodblock, Edo, ink wash
Fantasy:      epic, dramatic, ethereal, mystical, painterly
Cute:         kawaii, ghibli, whimsical, soft, warm
Tech:         cyberpunk, neon, holographic, circuit, chrome
```

### Negative Prompt Starter

```
bad anatomy, blurry, low quality, watermark, text,
signature, extra limbs, deformed, ugly
```

---

*"The image speaks to those who cannot read the code."*

## Sources

- [Evo-Ukiyoe-v1](https://huggingface.co/SakanaAI/Evo-Ukiyoe-v1)
- [ukiyo-e-art LoRA](https://huggingface.co/KappaNeuro/ukiyo-e-art)
- [sdxl_chinese_ink_lora](https://huggingface.co/ming-yang/sdxl_chinese_ink_lora)
- [Pony Diffusion V6 XL](https://openlaboratory.ai/models/pony-diffusion-v6-xl)
- [Best SD Fantasy Models](https://shakersai.com/ai-tools/images/stable-diffusion/stable-diffusion-models-for-fantasy/)
- [Prompt Engineering for SD](https://portkey.ai/blog/prompt-engineering-for-stable-diffusion)
