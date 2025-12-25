# Last Session: ATC Experiment Mode

**Date:** 2025-12-25 (Christmas!)
**Previous:** clood proxy & Issue #190

---

## WHAT WE BUILT

### ATC Experiment Mode (for tracking multi-step agent challenges)

New dashboard mode for tracking iterative coding experiments like Chimborazo rebuild and Bonsai Garden Doom.

```bash
clood atc --mode experiment --port 8080
```

**Hierarchical Event Structure:**
```
Session (e.g., "Chimborazo Rebuild Session 1")
  └── Step 1: "Analyze Strata thoreau module"
        ├── Iteration 1: qwen2.5-coder:3b @ mac-laptop
        ├── Iteration 2: qwen2.5-coder:7b @ mac-mini
        └── Validation: go build ./... [PASS]
  └── Step 2: "Implement HTTP fetcher"
        └── ...
```

**New Endpoints:**
- `POST /experiment` - Receive hierarchical events
- `GET /sessions` - Get current session state

**Event Types:**
- `session_start` / `session_end`
- `step_start` / `step_end`
- `iteration_start` / `iteration_end`
- `validation` (with pass/fail/skip status)

### GitHub Issues Created

- **#191** - Chimborazo Rebuild via Agent Swarm
  - Clean-room rebuild of geospatial toolkit
  - Agents analyze Strata Python, rebuild in Go from scratch
  - No peeking at existing chimborazo code
  - Success = SVG output matches Strata

- **#192** - Bonsai Garden Doom Demo
  - Silver tier: 3 rooms, 12 bonsai, 3 textures
  - Canvas 2D raycasting, billboard sprites
  - Uses `clood bonsai -f svg` for trees
  - Uses `rat-king fill` for wall textures

---

## FILES CHANGED

**Modified:**
- `internal/commands/atc.go`
  - Added `ExperimentSession`, `ExperimentStep`, `ExperimentIteration`, `ValidationResult`, `ExperimentEvent` structs
  - Added session tracking to Hub (`sessions` map, mutex protection)
  - Added `/experiment` endpoint for hierarchical events
  - Added `/sessions` endpoint for session state
  - Added `atcExperimentHTML` - full dashboard with timeline, progress bar, iterations display
  - Added helper functions: `getString`, `getInt`, `getFloat`

---

## DASHBOARD FEATURES

The experiment mode HTML includes:
- **Header** with connection status
- **Session panel** with:
  - Session name and progress (Step X/Y)
  - Progress bar
  - Timeline with step cards
  - Iteration rows (model, host, stats)
  - Validation results (command, output, errors)
- **Sidebar** with:
  - Hosts panel (online/offline/busy status)
  - Event feed (last 50 events)

Visual states:
- Running: green border, pulse animation
- Completed: blue border
- Failed: red border
- Pending: gray

---

## TESTING

```bash
# Start ATC in experiment mode
clood atc --mode experiment --port 8080

# Send session start
curl -X POST http://localhost:8080/experiment \
  -H "Content-Type: application/json" \
  -d '{
    "type": "session_start",
    "session_id": "chimborazo-001",
    "data": {"name": "Chimborazo Rebuild", "total_steps": 5}
  }'

# Send step start
curl -X POST http://localhost:8080/experiment \
  -H "Content-Type: application/json" \
  -d '{
    "type": "step_start",
    "session_id": "chimborazo-001",
    "step_id": "step-1",
    "data": {"number": 1, "name": "Analyze Strata thoreau module"}
  }'

# Send iteration start
curl -X POST http://localhost:8080/experiment \
  -H "Content-Type: application/json" \
  -d '{
    "type": "iteration_start",
    "session_id": "chimborazo-001",
    "step_id": "step-1",
    "data": {"number": 1, "step": 1, "model": "qwen2.5-coder:3b", "host": "mac-laptop"}
  }'

# Check session state
curl http://localhost:8080/sessions | python3 -m json.tool
```

---

## EXPERIMENT PLANS

### Chimborazo Rebuild (#191)
- **Goal:** Agents rebuild geospatial toolkit from Strata Python analysis
- **Strategy:** Options B+C (steer agents, don't skip hard problems)
- **Model escalation:** qwen:3b -> qwen:7b -> deepseek-r1:8b -> Opus 4.5
- **Validation:** SVG output comparison with Strata reference

### Bonsai Garden Doom (#192)
- **Goal:** First-person 3D bonsai garden exploration
- **Target:** Silver tier (3 rooms, 12 bonsai, 3 textures)
- **Approach:** Billboard sprites, Canvas 2D raycasting, MVP focus
- **Assets:** `clood bonsai -f svg`, `rat-king fill`

Both experiments can span multiple sessions with clear instructions and end states.

---

## NEXT STEPS

1. Run experiments on Christmas morning
2. Watch ATC dashboard for real-time progress
3. Compare Chimborazo rebuild to existing feature branches
4. Iterate on Bonsai Garden until playable

---

```
Snow falls on the tower,
Agents rebuild through the night—
Christmas code awakes.
```
