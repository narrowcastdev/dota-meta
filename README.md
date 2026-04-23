# dota-meta

Bracket-specific Dota 2 meta intelligence. Fetches hero stats from [STRATZ](https://stratz.com), computes tier quadrants per bracket, week-over-week deltas, climb tags, and momentum trends, and emits a Reddit post + static site.

## Install

```bash
go install github.com/narrowcastdev/dota-meta/cmd/dota-meta@latest
```

Or build from source:

```bash
git clone https://github.com/narrowcastdev/dota-meta.git
cd dota-meta
go build -o dota-meta ./cmd/dota-meta
```

## Running

STRATZ requires a bearer token. Get one from <https://stratz.com/api> and export it:

```bash
export STRATZ_TOKEN="…"
```

Then run the full pipeline:

```bash
./update.sh                  # builds, archives previous snapshot, fetches fresh data, generates reddit.md
DOTA_PATCH=7.40c ./update.sh # bump the patch string (defaults to value in update.sh)
```

Or invoke directly:

```bash
dota-meta                            # Reddit post to stdout
dota-meta --json                     # raw analysis JSON
dota-meta --html --patch 7.40b       # write docs/data.json for static site
dota-meta --output reddit.md         # write Reddit post to file
dota-meta --min-picks 2000           # raise qualification threshold
```

## What it analyzes

Five brackets (Herald-Guardian, Crusader-Archon, Legend-Ancient, Divine, Immortal). Heroes need ≥1000 picks per bracket to qualify.

- **Tier quadrants** — per-bracket 2×2 split of WR vs PR medians → Meta Tyrant / Pocket Pick / Trap / Dead. Cores and Supports classified separately so support-pool medians don't drag carry-pool classifications.
- **Climb tag** — `scales-up` / `scales-down` / `universal` based on Herald-Guardian vs Divine WR delta (±2pp threshold).
- **Momentum** — `rising` / `falling-off` / `hidden-gem` / `dying` from the last 4 weekly WR/PR points via least-squares slope. Noise floor of 1.0pp across the 4-week projection.
- **Deltas** — WR/PR change vs prior snapshot (if one exists in `docs/history/`).

## Snapshot history

Each `update.sh` run archives the previous `docs/data.json` into `docs/history/data-YYYY-MM-DD.json`. `dota-meta` auto-loads the newest prior snapshot to compute WR/PR deltas. Safe to commit the history directory — one file per run, ~tens of KB.

## Static site

Hosted at [dota.narrowcast.dev](https://dota.narrowcast.dev). Reads `docs/data.json`. To publish:

```bash
./update.sh
git add docs/data.json docs/history reddit.md
git commit -m "data: update hero stats"
git push
```

## Data source

STRATZ GraphQL (`api.stratz.com/graphql`). One `constants.heroes` call for the catalog plus eight `heroStats.winWeek` calls (one per rank bracket) per run. Rate limit: 7 rps / 132 rpm — we sleep 200ms between bracket calls.

OpenDota client archived under `internal/api/opendota/` (no longer used by the default pipeline).

## License

MIT

---

Data from [STRATZ](https://stratz.com). Built by [Narrowcast](https://github.com/narrowcastdev).
