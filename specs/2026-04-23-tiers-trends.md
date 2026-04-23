# Spec — Tier quadrants, bracket-climb tags, trend layer

**Date:** 2026-04-23
**Status:** Draft
**Scope:** dota-meta analysis output + frontend

## Problem

Current output is a stats dump: Best / Sleepers / Traps / Supports tables sorted by win rate. Gamers don't draft from raw WR. They want:

1. **Pick guidance** — ban this, first-pick that, last-pick counter, skip.
2. **Bracket context** — does this hero scale up with skill or stomp lower brackets?
3. **Meta direction** — rising, falling, stable? Is it worth learning now?

## Goals

- Replace flat WR tables with a 2×2 quadrant tier system per bracket.
- Tag every hero with a bracket-climb label (`scales up` / `scales down` / `universal`).
- Start collecting weekly snapshots; expose WR/PR deltas and a momentum tag once enough data exists.

## Non-goals

- Item/skill builds (r/DotA2 readers explicitly said no).
- Synergy/counter matchups (future work — STRATZ has the endpoint, defer).
- ML forecasting. Deltas and slope only.

## Data source

**Primary: STRATZ GraphQL** (`api.stratz.com/graphql`, bearer token).

Why switch from OpenDota:
- `heroStats.winWeek` returns 20 weeks of history per hero per bracket. Trends available immediately, no snapshot accumulation wait.
- 8 brackets incl. **Immortal as separate bucket** (OpenDota bracket 8 is empty). Top-MMR data is where meta is sharpest.
- One query per bracket returns all heroes × all weeks (~2500 rows). 8 requests total per refresh. Well under 7rps / 132rpm limit.
- Rich extras for future use: item purchase, matchups, talents, lane outcome, ban rate — no need to swap source again.
- Token valid until 2027-04-17 per JWT `exp`.

**Fallback: OpenDota** (kept, not run by default).
- Archive existing `internal/api/` as `internal/api/opendota/`. Keep intact, no changes.
- STRATZ lives at `internal/api/stratz/`.
- Do **not** blend numbers — bracket definitions differ (OpenDota "Divine" bundles Divine+Immortal; STRATZ separates them). Mixing confuses the reader.
- Fallback trigger (future): STRATZ 5xx / rate-limit / token error → run OpenDota path, tag output `source: opendota` in UI. Not implemented in step 1; safety net only.

Snapshot storage:
- `docs/history/stratz-YYYY-MM-DD.json` per weekly run (STRATZ raw response, per bracket concatenated).
- Already 20wk of past data embedded in today's first fetch — backfill at migration time, no need to wait.

---

## Design

### 1. Quadrant tiers (per bracket)

For each bracket, classify each qualified hero into one of four tiers using median WR and median PR as splits:

| Tier | Rule | Meaning |
|------|------|---------|
| **Meta Tyrant** | WR ≥ median WR **AND** PR ≥ median PR | Ban or first-pick. Popular *and* winning. |
| **Pocket Pick** | WR ≥ median WR **AND** PR < median PR | Sleeper. Last-pick counter, high value per pick. |
| **Trap** | WR < median WR **AND** PR ≥ median PR | Popular but losing. Stop drafting on vibes. |
| **Dead** | WR < median WR **AND** PR < median PR | Ignored, correctly. Skip from UI. |

**Why median, not fixed thresholds (53% WR etc.):** meta shifts. Patch could push baseline WR up or down. Medians self-adjust.

**Split bands by role family** (separate quadrants for cores vs supports):
- Core: any hero *not* tagged `Support` only.
- Support: tagged `Support` and not `Carry` (keeps flex heroes like Wraith King in cores).

Two quadrants per bracket × 4 brackets = 8 tables. Compact grid view in UI.

**Min-picks floor:** 1000 per bracket (current default). Below that, ignore — skill groups have wildly different sample sizes and noise dominates.

### 2. Bracket-climb tag

For each hero with ≥ 1000 picks in both Herald-Guardian and Divine:

- `delta = divineWR − heraldWR`
- **Scales up** — `delta ≥ +2.0pp`. Needs mechanics. Invoker, Visage, Meepo.
- **Scales down** — `delta ≤ −2.0pp`. Stomps lower brackets, punished by skill.
- **Universal** — `|delta| < 2.0pp`. Works everywhere.

Display as a single column next to hero name in every bracket's table.

### 3. Trend layer

**Snapshot store:** `docs/history/data-YYYY-MM-DD.json`. `update.sh` copies `data.json` into `history/` with today's date before writing the fresh file.

**Keep all snapshots** — they're ~128KB each, git LFS not needed. After a year, re-evaluate.

**Tag each snapshot with patch version.** Dota 2 patch = discontinuity; trends shouldn't cross patches. Add a `"patch": "7.40b"` field to `data.json`. Source: OpenDota `/constants/patch` or manual bump in `update.sh`.

**Deltas (2+ snapshots available):**
- `wrDelta = currentWR − priorWR` (same bracket, same hero)
- `prDelta = currentPR − priorPR`
- Show as ±N.Np next to WR/PR in UI.

**Momentum tag (4+ snapshots in same patch cycle):**
- Fit linear regression on last 4 weekly WR values.
- Slope × 4 = projected 4-week change.

| Tag | Rule |
|-----|------|
| 🔥 Rising star | WR slope ↑ **AND** PR slope ↑ |
| ⚠️ Falling off | WR slope ↓ **AND** PR slope ↑ (hype bubble) |
| 💤 Hidden gem | WR slope ↑ **AND** PR slope flat/↓ |
| 📉 Dying | WR slope ↓ **AND** PR slope ↓ |
| — | Otherwise (flat / mixed below noise floor) |

**Noise floor:** |slope × 4| < 1.0pp → no tag. Don't cry wolf on noise.

**Patch reset:** momentum buffer resets on patch change. Show "new patch — trend data building" until 4 in-patch snapshots exist.

---

## Data model changes

`analysis.go`:

```go
type Tier int
const (
    TierMetaTyrant Tier = iota
    TierPocketPick
    TierTrap
    TierDead
)

type ClimbTag string
const (
    ClimbUp       ClimbTag = "scales up"
    ClimbDown     ClimbTag = "scales down"
    ClimbUniversal ClimbTag = "universal"
)

type MomentumTag string // "rising", "falling-off", "hidden-gem", "dying", ""

type HeroStat struct {
    // existing fields...
    Tier        Tier
    ClimbTag    ClimbTag
    WRDelta     *float64 // nil if no prior snapshot
    PRDelta     *float64
    Momentum    MomentumTag
    WRHistory   []float64 // last 8 weekly WRs for sparkline, most-recent last
}

type BracketAnalysis struct {
    // existing fields...
    Cores    []HeroStat // replaces Best/Sleepers/Traps
    Supports []HeroStat
    // Tiers derived at render time from Cores/Supports
}
```

Drop `Best`, `Sleepers`, `Traps` once frontend migrates.

`FullAnalysis`:

```go
type FullAnalysis struct {
    // existing...
    Patch           string
    SnapshotDate    time.Time
    PriorSnapshot   *time.Time // nil if first run
    SnapshotsInPatch int       // how many in current patch cycle
}
```

## Frontend changes

`docs/app.js` + `docs/index.html`:

- Per bracket: two 2×2 grids (cores, supports). Each cell = tier label + hero chips.
- Hero chip: `[name] [climb-tag-icon] [WR%] [±delta] [PR%] [momentum-icon]`.
- Hover chip → sparkline of last 8 weekly WRs.
- Filter: bracket selector (keep existing), role selector (cores / supports / all).
- Legend explaining tiers + climb tags + momentum icons.

Render "Dead" tier as a collapsed section — title only, click to expand. Keeps view focused.

## Implementation order

1. **Quadrant tiers** — `analysis.go` computes Tier per hero. Update `format/` to emit tier instead of Best/Sleepers/Traps. Update frontend to render grids. *Ship with current single snapshot.*
2. **Bracket-climb tag** — extend existing `analyzeBracketDelta` to tag every hero, not just top/bottom 5. Add `ClimbTag` to `HeroStat`.
3. **History store** — modify `update.sh` to copy `data.json` → `history/data-YYYY-MM-DD.json` before regenerating. Add `patch` field, hand-bumped for now.
4. **Deltas** — on analysis run, load prior snapshot if exists, compute WR/PR deltas per hero per bracket. Display in UI.
5. **Momentum** — requires 4 in-patch snapshots. Ship logic, gate UI behind `snapshotsInPatch >= 4`. Show "trend data building" placeholder meanwhile.
6. **Sparklines** — smallest SVG inline per chip, no library. Last 8 weekly WRs.

Steps 1-2 ship this weekend from current data. Step 3 activates next `update.sh` run. Steps 4-6 unlock over the following weeks as history fills.

## Reddit post format impact

`reddit.md` currently lists top-5 per category. Rewrite as:
- Per bracket: "Ban list" (top 3 Meta Tyrants), "Pocket picks" (top 3 Pocket), "Stop picking" (top 3 Traps).
- "Rising this week" section once momentum data exists (step 5+).
- Keep under existing length — r/DotA2 reader feedback was "readable, not dense."

## Open questions

- Median-split across *all* qualified heroes per bracket, or separate medians for cores vs supports? **Default: separate medians** — otherwise most supports land below core median WR and fill Trap/Dead unfairly.
- Patch version source: OpenDota `/constants/patch` vs manual. **Default: manual in `update.sh` for now** — one string per week is cheap; automate if it becomes annoying.
- Store history in-repo or separate branch? **Default: in-repo `docs/history/`** — simple, git tracks it, CF Pages can serve if needed later.
- Sparkline: cores only, or all tiers? **Default: all tiers that render** (skip Dead).

## Success check

- A Divine reader can see "Visage: Pocket Pick, scales up, 💤 Hidden gem" at a glance.
- A Herald reader sees Wraith King as Meta Tyrant + scales-down — knows it's a crutch to wean off.
- Weekly `update.sh` produces one new snapshot; momentum appears within 4 weeks of shipping step 3.
