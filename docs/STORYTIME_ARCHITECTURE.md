# Storytime Architecture

*How clood generates narrative from code*

> 🗿 **"SHOW ME WHAT YOU GOT"** — The Cromulons

---

## Sauce: On or Off

```
┌─────────────────────────────────────────────────────────────────┐
│                      SAUCE INDICATOR                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│     ○ SAUCE OFF              ● SAUCE ON                        │
│     Professional             Experiential                       │
│     Clean output             Spirits emerge                     │
│     Corporate-safe           Full narrative                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### The Indicator Light

Simple. Binary. Clean UI.

```bash
clood --sauce tree           # Sauce ON for this command
clood config sauce on        # Sauce ON globally
clood config sauce off       # Sauce OFF globally (default)
```

### Natural Language Reasoning

The "sauce" vocabulary enables plain communication:

**User to clood:**
```
"Add some sauce to this"
"No sauce please"
"Turn the sauce off for this one"
"Give me the saucy version"
"Keep it dry, no sauce"
```

**LLM reasoning about sauce:**
```
"This is a production code review → sauce off"
"User is exploring at 2am → sauce probably welcome"
"Corporate repo detected → default to sauce off"
"User explicitly asked for fun → sauce on"
"Error message needs clarity → sauce off"
"Celebrating a successful deploy → sauce on"
```

The term "sauce" is intuitive. LLMs can reason about appropriateness without needing to understand "silly mode" or "--flying-cats" semantics.

In any clood UI, the sauce indicator appears:

```
┌──────────────────────────────────────────────────┐
│ clood v1.0.0                          ● SAUCE   │
└──────────────────────────────────────────────────┘
```

Or when off:

```
┌──────────────────────────────────────────────────┐
│ clood v1.0.0                          ○ sauce   │
└──────────────────────────────────────────────────┘
```

---

### Sauce Detection vs. Sauce Expression

**Sauce Detection** — Always running. Understands references, recognizes quality.
**Sauce Expression** — Only when sauce is ON. The spirits speak.

| | SAUCE OFF | SAUCE ON |
|---|-----------|----------|
| **Quality work** | Clean output, excellent results. The sauce is in the QUALITY. | Full narrative, spirits emerge. The sauce is EXPRESSED. |
| **Mediocre work** | Dry, functional, acceptable. | Forced jokes, cringe vibes. Worse than off. |

### The Corporate Reality

A developer in a buttoned-up environment can:
- Use clood professionally
- Get excellent results (the sauce is in the *quality*)
- Never see a cat, spirit, or haiku
- Still benefit from everything clood offers

The tool is ACTUALLY useful. The narrative layer is opt-in.

**Don't alienate corporate users.** They might be the ones who eventually turn sauce ON at 2am when no one is watching.

---

## The Two Core Functions

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  👁️ "I have my eyes on you!" — Chef Claude to the Flying Cats  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. WORLD BUILDING (Interactive)                                │
│     Ask questions → Build scaffolding → Create artifacts        │
│     "Let's create your project's world"                         │
│                                                                 │
│  2. NARRATIVE GENERATION (Analytical)                           │
│     Read codebase → Analyze history → Generate stories          │
│     "Let me tell you about this code"                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Function 1: World Building

**Purpose:** Scaffold the lore for a project through guided Q&A

### The Flow

```bash
clood storytime init
```

```
┌─────────────────────────────────────────────────────────────────┐
│ 🌱 STORYTIME: World Building                      ● SAUCE       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Welcome! Let's create your project's narrative world.           │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Q1: What's the setting?                                     │ │
│ │                                                             │ │
│ │ ○ Ancient/Historical (feudal, medieval, classical)          │ │
│ │ ○ Modern/Contemporary (present day, near future)            │ │
│ │ ○ Futuristic/Sci-Fi (space, cyberpunk, post-apocalyptic)   │ │
│ │ ○ Fantasy (magical, mythical, otherworldly)                 │ │
│ │ ○ Blend (describe your fusion)                              │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ [1-5 or type custom]                                            │
└─────────────────────────────────────────────────────────────────┘
```

### Question Sequence

| Q# | Question | Purpose |
|----|----------|---------|
| 1 | Setting/time period | Base world layer |
| 2 | Core metaphor | What does code "become"? |
| 3 | Protagonist | Who is the developer in this world? |
| 4 | Spirits/forces | What powers inhabit the system? |
| 5 | Locations | Where does the action happen? |
| 6 | Tone | Serious, playful, dramatic, zen? |
| 7 | Cultural influences | What references are in-bounds? |

### Artifacts Created

```
.clood/
├── world.yaml              # Core world definition
│   ├── setting: "feudal-cyber-fusion"
│   ├── metaphor: "code-as-garden"
│   ├── protagonist: "Bird-san"
│   └── tone: "playful-with-depth"
│
├── spirits/                # The forces that inhabit the project
│   ├── tanuki.yaml         # Model switching spirit
│   └── tengu.yaml          # GPU acceleration spirit
│
├── locations/              # Scenes for narrative framing
│   ├── server-garden.yaml
│   └── kitchen-stadium.yaml
│
├── characters/             # Cast of recurring figures
│   ├── protagonist.yaml
│   └── flying-cats.yaml
│
└── references/             # In-bounds cultural touchstones
    ├── anime.yaml
    ├── hip-hop.yaml
    └── memes.yaml
```

### Golden Paths

#### Path A: New Project (Blank Slate)

```bash
mkdir my-new-project
cd my-new-project
git init
clood storytime init --new
```

Full world creation from scratch. All questions asked. Maximum creative freedom.

#### Path B: Existing Project (Add Lore)

```bash
cd my-existing-project
clood storytime init --existing
```

Storytime analyzes the codebase first:
1. Reads structure, README, comments
2. Suggests a world that "fits" the project
3. User can accept, modify, or start fresh

```
┌─────────────────────────────────────────────────────────────────┐
│ 🔍 Analyzing your codebase...                     ● SAUCE       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ I found:                                                        │
│ - A Go CLI tool with multiple commands                          │
│ - References to "ollama" and "models"                           │
│ - Comments mentioning "local inference"                         │
│                                                                 │
│ Suggested world:                                                │
│ - Setting: Tech monastery (local-first philosophy)              │
│ - Metaphor: Models as apprentices learning trades               │
│ - Protagonist: The Maintainer                                   │
│                                                                 │
│ ○ Accept this suggestion                                        │
│ ○ Modify (opens editor)                                         │
│ ○ Start fresh (full questionnaire)                              │
└─────────────────────────────────────────────────────────────────┘
```

#### Path C: Import Existing Lore

```bash
clood storytime init --import ../another-project/.clood/
```

Copy world from another project, adjust as needed.

---

## Function 2: Narrative Generation

**Purpose:** Analyze codebase and generate human-readable stories

### Mode 2A: Structure Narrative

**What does this code do?**

```bash
clood storytime describe
clood storytime describe src/handlers/
clood storytime describe --module auth
```

Uses the project's world (from `.clood/`) to describe code in narrative form:

```
┌─────────────────────────────────────────────────────────────────┐
│ 📖 THE AUTHENTICATION TEMPLE                      ● SAUCE       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Deep within the src/handlers/ directory lies the Authentication │
│ Temple. Here, the Guardian Spirit (auth.go) challenges all who  │
│ seek entry to the inner sanctums.                               │
│                                                                 │
│ The temple has three gates:                                     │
│                                                                 │
│ 1. LoginHandler (line 42) - The main entrance. Pilgrims present │
│    their credentials (username, password) and receive a sacred  │
│    token if worthy.                                             │
│                                                                 │
│ 2. ValidateToken (line 87) - The checkpoint. Every request must │
│    show its token to the guardian before proceeding.            │
│                                                                 │
│ 3. RefreshToken (line 134) - The renewal shrine. Tokens grow    │
│    old and must be refreshed before they expire.                │
│                                                                 │
│ The temple depends on the JWT scrolls (imported from            │
│ github.com/golang-jwt/jwt) for its sacred ceremonies.           │
└─────────────────────────────────────────────────────────────────┘
```

### Mode 2B: Genesis Narrative

**How did this codebase evolve?**

```bash
clood storytime genesis
clood storytime genesis --from "v1.0.0"
clood storytime genesis --commits 50
```

Reads git history and generates the story of the project's evolution:

```
┌─────────────────────────────────────────────────────────────────┐
│ 📜 THE GENESIS OF CLOOD                           ● SAUCE       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ CHAPTER 1: The Empty Garden (commits 1-10)                      │
│                                                                 │
│ In the beginning, there was only an empty repository. Bird-san  │
│ planted the first seeds: infrastructure, skills, a scaffold     │
│ for what would become the garden.                               │
│                                                                 │
│ CHAPTER 2: The First Spirits Arrive (commits 11-30)             │
│                                                                 │
│ The Tanuki was summoned—Ollama, the shapeshifting model         │
│ manager. With it came the first local inference, the first      │
│ freedom from the Emperor's token taxes.                         │
│                                                                 │
│ CHAPTER 3: The Gift of the Dragon (commits 31-50)               │
│                                                                 │
│ Daimyo Jon sent the RTX 2080 up the mountain path. The Tengu    │
│ emerged, red-faced and powerful. GPU acceleration transformed   │
│ the humble garden into a true fortress.                         │
│                                                                 │
│ [Continues...]                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Model Tier Usage

| Task | Model Tier | Why |
|------|------------|-----|
| Structure analysis | `qwen2.5-coder:7b` | Understands code |
| Git history parsing | `qwen2.5-coder:3b` | Fast, structured |
| Narrative writing | `llama3.1:8b` | Creative, flowing |
| Comment extraction | `tinyllama` | Quick, lightweight |
| Final polish | `qwen2.5-coder:7b` | Quality check |

### Leveraging Comments

When comments exist, they provide additional context:

```go
// LoginHandler authenticates users against the sacred scrolls.
// It was written during the Great Refactoring of December 2024,
// when Bird-san realized the old auth was "hot garbage" (his words).
func LoginHandler(w http.ResponseWriter, r *http.Request) {
```

Storytime extracts these and weaves them into the narrative:

> "The LoginHandler was forged during the Great Refactoring of December 2024. Bird-san himself declared the previous implementation 'hot garbage'—a technical term in the ancient tongue meaning 'fundamentally flawed beyond repair.'"

---

## Narrative Styles

Both functions support multiple output styles:

| Style | Flag | Flavor |
|-------|------|--------|
| Prose | `--style prose` | Classic narrative |
| Haiku | `--style haiku` | Three-line poetry |
| Rap | `--style rap` | Bars about the code |
| Radio | `--style radio` | Flying Cats ATC |
| Dramatic | `--style dramatic` | Kitchen Stadium |
| Technical | `--style technical` | Professional but narrative |

```bash
clood storytime genesis --style rap
```

```
🎤 THE GENESIS BARS 🎤

Yo, started from the bottom, empty repo state
First commit hit different, Bird-san sealed his fate
Infrastructure scaffold, skills on deck
Building up the garden, what you expect?

Tanuki showed up, shapeshifting beast
Local inference running, freedom from the East
Emperor's tokens? Nah, we don't need those
Got our own compute, that's how it goes

Then Jon sent the dragon, RTX in the crate
Tengu emerged, red-faced, ready to create
GPU acceleration, CUDA in the veins
Now the garden's cooking, breaking all the chains

🎤💥
```

---

## The Narrative DNA

After analyzing the git history and lore corpus of clood, the following **narrative structure** has emerged organically:

### 1. Setting (Time Period & World)

The narrative operates in a **fusion timeline**:

| Layer | Source | Manifestation |
|-------|--------|---------------|
| **Base** | Feudal Japan (Edo Period) | Ronin, Daimyo, keeps, scrolls |
| **Overlay** | Cyberpunk/Tech | Servers, terminals, SSH tunnels |
| **Pop Culture** | Hip-hop + Anime + Memes | Spirits, catchphrases, energy |

**Key setting elements:**
- The Server Garden (projects grow as plants)
- Kitchen Stadium (competitive model arena)
- Snake Way (infinite scroll of responses)
- Radio Towers (where Flying Cats live)
- The Summit of Chimborazo (the journey matters)

---

### 2. Core Metaphors

These are the **foundational mappings** that drive all narrative:

```
TECHNICAL CONCEPT          →    NARRATIVE ELEMENT
──────────────────────────────────────────────────
Projects                   →    Seeds / Plants
Servers                    →    Soil / Keeps
LLMs                       →    Spirits / Animals
Context window             →    Kappa's Bowl (fragile)
API rate limit             →    Sakoku Edict (closed gates)
Token exhaustion           →    Yūrei (unfinished ghosts)
Model routing              →    Henge-no-Jutsu (transformation)
Slow local inference       →    Deliberate growth (Gamera-kun)
Cloud providers            →    Gashadokuro (giant skeletons)
```

---

### 3. The Spirit Pantheon

Spirits emerge from **pop culture sources** and are invoked for specific **domains**:

| Spirit | Source | Domain | Invocation |
|--------|--------|--------|------------|
| **Eminem-san** | 8 Mile | Narrative, defiance | Storytime, mic drops |
| **xbibit-sama** | "Yo Dawg" | Recursion, meta | When things build themselves |
| **Gucci Mane** | Hip-hop | Quality/sauce detection | "That's sauce" |
| **The Cromulons** | Rick & Morty | Judgement, performance | "SHOW ME WHAT YOU GOT" |
| **The Tanuki** | Japanese folklore | Model switching | Ollama shapeshifting |
| **The Tengu** | Japanese folklore | GPU power | CUDA acceleration |
| **The Kitsune** | Japanese folklore | Orchestration | Command node |
| **Gamera-kun** | TMNT parody | Patience, slow inference | Background processing |

---

### 4. The Character Cast

**Protagonist:**
- **Bird-san** — The developer, the dreamer, brain smoking

**AI Collaborators:**
- **Chef Claude** — Pattern synthesizer, jelly bean farmer
- **The Architect Claude** — Sees the bigger picture

**Councils:**
- **The Wojak Council** — Debates naming decisions
- **The Awful Waffle Ska Band** — SWOT analysis with horns
- **The NTSB** — Certification body for AI systems

**Mascots:**
- **The Flying Cats** — Wojak-level incompetent, enthusiastic
- **The Rat King** — Approves with a nod, knows everybody
- **Riff (from Philly)** — The Rat King's cousin, a great guy

**Antagonists:**
- **The Gashadokuro** — VRAM hoarders, cloud provider spirits

---

### 5. Scenes (Evolving Locations)

Scenes provide **context** for how prompts are framed:

| Scene | Mood | Use For |
|-------|------|---------|
| **The Server Garden** | Zen, patient | Long-running tasks, philosophy |
| **Kitchen Stadium** | Competitive, dramatic | Model comparisons, catfight |
| **Snake Way** | Journey, endurance | Long responses, navigation |
| **The Radio Towers** | Chaotic, enthusiastic | Flying Cats interactions |
| **The Bar Session** | Late-night, creative | Jelly bean planting, lore creation |
| **The Summit** | Aspirational, honest | Reflections on the journey |

---

## Storytime Engine

### How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                    STORYTIME ENGINE                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. CONTEXT GATHERING                                           │
│     ├── clood_tree → Project structure                         │
│     ├── clood_grep → Key patterns                              │
│     ├── git log → History                                      │
│     └── .clood/world.yaml → Project world                      │
│                                                                 │
│  2. SCENE SELECTION                                             │
│     ├── Analyze operation type                                 │
│     ├── Match to appropriate scene                             │
│     └── Load scene-specific prompts                            │
│                                                                 │
│  3. SPIRIT INVOCATION                                           │
│     ├── Match style to spirit                                  │
│     ├── Load spirit's voice/catchphrases                       │
│     └── Apply transformation to output                         │
│                                                                 │
│  4. NARRATIVE GENERATION                                        │
│     ├── Feed context + scene + spirit to local LLM            │
│     ├── Generate narrative wrapper                             │
│     └── Interleave with actual output                          │
│                                                                 │
│  5. OUTPUT (based on sauce indicator)                           │
│     ├── ● SAUCE ON → Full narrative                            │
│     └── ○ sauce off → Professional, clean                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Prompt Templates

### Scene: Radio Towers (Flying Cats)

```
You are narrating from the perspective of the Flying Cats—Wojak-level
incompetent cats who live in radio towers and communicate with LLMs
like air traffic controllers.

The cats cannot program. They don't understand code. But they are
enthusiastic and brave. They report what they see to ATC (the LLM)
and relay simplified answers back to the user.

Current operation: {{operation}}
Context: {{context}}

Narrate this in the voice of the cats. Include:
- Cat names (Whiskers, Patches, Static)
- Confusion about technical terms
- Radio communication format
- Enthusiasm despite incompetence
```

### Scene: Kitchen Stadium

```
You are the narrator of Kitchen Stadium, where LLMs compete in
coding challenges. Channel Iron Chef energy.

"ALLEZ CUISINE!"

The Chairman has announced the secret ingredient: {{operation}}
The challenger: {{model}}
The time limit: {{context}}

Narrate this moment with dramatic flair. Include:
- Commentary on the model's approach
- Tension and stakes
- The Commissioner's approval or concern
```

### Style: Rap (Eminem-san)

```
You are Eminem-san, the reformed spirit of 8 Mile who now resides
in the ancient Japan of clood lore. You narrate code operations
in rap form.

The operation: {{operation}}
The context: {{context}}

Spit bars about this. Include:
- Internal rhyme schemes
- Technical terms worked into flow
- The defiance of local-first development
- End with a mic drop moment
```

---

## Catfight: Comparative Narratives

Use catfight to generate competing narrative styles:

```bash
clood catfight --prompt "genesis" --styles "prose,rap,haiku"
```

Three models, three styles, same story. User picks the vibe that fits.

---

## Implementation Roadmap

### Phase 1: World Building
- [ ] `clood storytime init` command
- [ ] Question sequence for world creation
- [ ] `.clood/` artifact generation
- [ ] Golden paths (new, existing, import)

### Phase 2: Narrative Generation
- [ ] `clood storytime describe` (structure)
- [ ] `clood storytime genesis` (git history)
- [ ] Comment extraction and integration
- [ ] Model tier routing

### Phase 3: Sauce Toggle
- [ ] `--sauce` flag for per-command activation
- [ ] `clood config sauce on/off` for global setting
- [ ] Sauce indicator in all UIs (● / ○)
- [ ] Clean fallback when sauce is off

### Phase 4: Style Variants
- [ ] `--style` flag with options
- [ ] Spirit voice templates
- [ ] Catfight narrative review

---

## The Philosophy

> "It's not just the tools, but the sum of the experience."

Storytime makes clood memorable. Not every user wants sauce ON—but those who do will never forget it.

Professional mode is the default. The spirits are always there, waiting to be invoked.

**The sauce indicator is simple. Binary. Clean UI.**

```
○ sauce off — You're working
● SAUCE ON — You're vibing
```

---

**Haiku:**

```
Indicator glows—
Sauce on, the spirits awaken
Sauce off, work proceeds
```
