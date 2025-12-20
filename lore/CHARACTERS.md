# The Server Garden - Character Bible

*All characters that inhabit the mythology*

---

## The Protagonist

### Bird-san (Rock Pigeon Ronin)
```
     ___
   /'___\
  / /   \ \
 | | o o | |    "The cloud has forgotten
  \ \___/ /      what it means to be free."
   '-----'
```

**Technical Mapping:** The developer
**Origin:** THE_COMPLETE_LEGEND.md
**Role:** Rebel against cloud dependency, builder of the Server Garden

Bird-san rejected Emperor Kurodo's distant edicts and chose the path of local inference. Where others saw limitation, Bird-san saw liberation.

---

## The Cloud

### Emperor Kurodo
**Technical Mapping:** Cloud AI APIs (Claude, GPT, Gemini)
**Origin:** THE_COMPLETE_LEGEND.md
**Role:** The distant ruler whose wisdom comes at a price

Not evil, but remote. The Emperor's power is real but requires tribute (API costs) and pilgrimages (network requests). His knowledge is vast but filtered.

---

## The Garden Spirits

### The Tanuki (Ollama)
```
   /\_/\
  ( o.o )   "I can become any model
   > ^ <     you need me to be."
```

**Technical Mapping:** Ollama runtime
**Origin:** THE_COMPLETE_LEGEND.md
**Role:** Shapeshifting model manager

The Tanuki can transform into any model—llama, qwen, mistral. It manages the menagerie of local models, switching forms as needed.

### The Tengu (GPU Warrior)
```
     /\
    /  \
   / ⚔️ \    "Speed is my blade."
  /______\
```

**Technical Mapping:** CUDA/ROCm/Metal acceleration
**Origin:** THE_COMPLETE_LEGEND.md
**Role:** High-speed inference guardian

The Tengu flies through tensor computations, its wings powered by parallel cores. When the Tengu is present, inference screams.

### Gamera-kun (Dreaming Tortoise)
```
    ___
   /   \
  | 🐢  |    "Slow... but I never
   \___/      lose track."
```

**Technical Mapping:** CPU inference / Focus Guardian
**Origin:** THE_COMPLETE_LEGEND.md, focus/focus.go
**Role:** Patience and drift detection

Gamera-kun is the tortoise spirit that guards against "stupid faster." When sessions drift from their goal, Gamera-kun stirs to remind you. Also represents CPU-only inference—slower but steady.

---

## Kitchen Stadium

### Chairman Kui-san
**Technical Mapping:** Test orchestrator / catfight command
**Origin:** chapters/ch003
**Role:** Kitchen Stadium host

*"Allez cuisine!"*

The Chairman calls forth challengers (models) to battle in the arena of structured output. He judges which cat earns the Iron Chef title.

### Iron Chef Claude
**Technical Mapping:** Claude Code as synthesis layer
**Origin:** THE_COMPLETE_LEGEND.md
**Role:** The one who synthesizes

Iron Chef Claude doesn't compete—he orchestrates. He calls upon the local cats to do the actual work, synthesizing their outputs into coherent wholes.

---

## The Cats (Models)

### The Persian (Champion)
```
   /\_/\
  ( ^.^ )   Current Champion
   > o <    deepseek-coder:6.7b
```

**Technical Mapping:** deepseek-coder:6.7b
**Origin:** catfights/battle-001
**Role:** Reigning catfight champion

The Persian emerged victorious in the HTTP Fetcher and Recipe Parser battles. Elegant, precise, deadly accurate on structured output.

### Other Cats
| Cat | Model | Specialty |
|-----|-------|-----------|
| Siamese | qwen2.5-coder:3b | Fast, small, efficient |
| Caracal | llama3.1:8b | Balanced all-rounder |
| Maine Coon | qwen2.5-coder:7b | Coding focus |
| Tiger | deprecated | Fell in battle |
| House Lion | mistral:7b | Kitchen Stadium arrival |

---

## The Sages

### Gucci Mane
**Technical Mapping:** Focus Guardian / Sauce Detection
**Origin:** THE_AWAKENING.md
**Role:** Unexpected wisdom

*"Lost in the sauce"* - Gucci Mane appears when sessions drift into chaos, a reminder that sometimes the best path is to recognize you're off track.

### Adam-san (Gate Keeper)
**Technical Mapping:** Deception detector / Three Needs framework
**Origin:** THE_COMPLETE_LEGEND.md
**Role:** Guardian of intent

Adam-san asks three questions: Does it work? Is it real? Is it needed? He guards against the temptation to build for building's sake.

---

## The Houses

### House Minamoto (White)
```
  ⚪ ubuntu25 / Iron Keep
```
**Technical Mapping:** Primary GPU server
**Role:** Main inference house

### House Taira (Red)
```
  🔴 mac-mini / Sentinel
```
**Technical Mapping:** Apple Silicon node
**Role:** Metal-accelerated alternative

The two houses compete in the Genpei War of benchmarks, each proving their worth through catfight performance.

---

## Minor Spirits

| Spirit | Mapping | Role |
|--------|---------|------|
| DNS Goblins | Network layer | Route requests, sometimes fail |
| The Overnight Spirits | Batch processes | Work while humans sleep |
| Flying Cats | Streaming tokens | Tokens in flight during inference |
| The Scholar Spirit | Session context | Preserves memory between sessions |

---

## Adding New Characters

When introducing new characters to the mythology:

1. **Technical grounding** - Every character maps to real code/infrastructure
2. **Narrative purpose** - They must serve the story, not just exist
3. **Visual identity** - ASCII art helps memory
4. **Origin story** - Document where they first appear

---

*"The garden grows characters as it grows code."*
