# Tier quadrants, climb tags, trend layer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace flat WR tables with 2×2 tier quadrants, climb-tags, snapshot history, deltas, momentum tags, and sparklines — driven by STRATZ GraphQL as the new primary data source.

**Architecture:** Keep Go pipeline `fetch → analyze → format → static site`. Add STRATZ client beside archived OpenDota client. `analysis` package gains `Tier`, `ClimbTag`, `MomentumTag`, history I/O, and linear-regression slope computation. `update.sh` snapshots `data.json` into `docs/history/` per run. Frontend renders grids + sparklines from one JSON blob.

**Tech Stack:** Go 1.26, STRATZ GraphQL (`api.stratz.com/graphql`, bearer token), vanilla JS/HTML/CSS in `docs/`, deployed via CF Pages.

**Repo root:** `/Volumes/Untitled/narrowcast/dota-meta`. All paths below are relative to this.

---

## File structure

**New:**
- `internal/api/stratz/client.go` — STRATZ GraphQL client + bearer auth.
- `internal/api/stratz/types.go` — response types (`HeroWeekStat`, `BracketData`).
- `internal/api/stratz/client_test.go` — parse golden fixture.
- `internal/api/stratz/testdata/response.json` — captured fixture.
- `internal/analysis/tier.go` — `Tier` enum, `ClassifyTiers` over cores/supports.
- `internal/analysis/tier_test.go`.
- `internal/analysis/climb.go` — `ClimbTag` enum + `TagClimb`.
- `internal/analysis/climb_test.go`.
- `internal/analysis/history.go` — load prior snapshot; compute WR/PR deltas.
- `internal/analysis/history_test.go`.
- `internal/analysis/momentum.go` — linear slope, `MomentumTag` + gating on `SnapshotsInPatch`.
- `internal/analysis/momentum_test.go`.

**Modified:**
- `internal/api/` → **move** existing files to `internal/api/opendota/` (package rename). Keep intact.
- `internal/analysis/analysis.go` — extend `HeroStat`, `BracketAnalysis`, `FullAnalysis`; wire tier/climb/momentum.
- `internal/format/json.go` — emit new fields; drop Best/Sleepers/Traps after frontend migrates.
- `internal/format/reddit.go` — rewrite sections: Ban list / Pocket picks / Stop picking / Rising.
- `cmd/dota-meta/main.go` — switch default data source to STRATZ, wire history load/save.
- `docs/index.html` + `docs/app.js` + `docs/style.css` — grids, chips, sparklines, legend.
- `update.sh` — copy `docs/data.json` → `docs/history/data-YYYY-MM-DD.json` before regen; accept/pass `--patch` flag.

**Kept but no changes:**
- `internal/api/opendota/*` (renamed archive).
- `testdata/herostats.json` (referenced by opendota tests).

---

## Conventions every task must follow

- Run `gofmt -s -l . ; go vet ./... ; go test ./...` before every commit. Code must be clean.
- Public repo — **no `Co-Authored-By` line** in commits. Default branch is `master`.
- No `npm`; yarn only for any JS work (none in this plan — plain static JS).
- STRATZ bearer token comes from env `STRATZ_TOKEN`. Never commit it.
- Use `t.Run` subtests and table-driven style where 3+ cases.
- Floats in tests: use `math.Abs(got-want) < 1e-6` or round. Don't compare raw.

---

## Task 1 — Archive OpenDota client as `internal/api/opendota`

**Files:**
- Move: `internal/api/api.go` → `internal/api/opendota/opendota.go`
- Move: `internal/api/api_test.go` → `internal/api/opendota/opendota_test.go`
- Modify: `internal/format/json.go`, `internal/analysis/analysis.go`, `cmd/dota-meta/main.go` — update imports.
- Modify: all `package api` → `package opendota`.

- [ ] **Step 1: Create the new directory and move files**

```bash
cd /Volumes/Untitled/narrowcast/dota-meta
mkdir -p internal/api/opendota
git mv internal/api/api.go internal/api/opendota/opendota.go
git mv internal/api/api_test.go internal/api/opendota/opendota_test.go
```

- [ ] **Step 2: Rename package and update test expectations if any**

In both moved files, replace the first line `package api` with `package opendota`.

- [ ] **Step 3: Update importing files**

For each file in `internal/analysis/analysis.go`, `internal/format/json.go`, `cmd/dota-meta/main.go`:
- Replace import path `"github.com/narrowcastdev/dota-meta/internal/api"` with `"github.com/narrowcastdev/dota-meta/internal/api/opendota"`.
- Replace package qualifier `api.Hero` → `opendota.Hero`, `api.FetchHeroStats` → `opendota.FetchHeroStats`, `api.ParseHeroStats` → `opendota.ParseHeroStats`.

- [ ] **Step 4: Run vet + tests**

```bash
go vet ./... && go test ./...
```
Expected: PASS (behaviour unchanged; rename only).

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: archive opendota client under internal/api/opendota"
```

---

## Task 2 — STRATZ GraphQL client: minimal fetch

**Files:**
- Create: `internal/api/stratz/types.go`
- Create: `internal/api/stratz/client.go`
- Create: `internal/api/stratz/client_test.go`
- Create: `internal/api/stratz/testdata/response.json`

The STRATZ query returns per-bracket `heroStats.winWeek` with up to 20 weekly entries per hero. One request per bracket (8 total). Rate limit: 7 rps / 132 rpm.

- [ ] **Step 1: Write `types.go` with response shape**

```go
package stratz

// Bracket is the STRATZ rank-bracket enum (see RankBracket in their schema).
type Bracket string

const (
	BracketHerald    Bracket = "HERALD"
	BracketGuardian  Bracket = "GUARDIAN"
	BracketCrusader  Bracket = "CRUSADER"
	BracketArchon    Bracket = "ARCHON"
	BracketLegend    Bracket = "LEGEND"
	BracketAncient   Bracket = "ANCIENT"
	BracketDivine    Bracket = "DIVINE"
	BracketImmortal  Bracket = "IMMORTAL"
)

// AllBrackets returns the 8 brackets in ascending order.
func AllBrackets() []Bracket {
	return []Bracket{
		BracketHerald, BracketGuardian, BracketCrusader, BracketArchon,
		BracketLegend, BracketAncient, BracketDivine, BracketImmortal,
	}
}

// HeroWeekStat is one weekly (hero, bracket) aggregate.
type HeroWeekStat struct {
	HeroID    int     `json:"heroId"`
	Week      int64   `json:"week"`      // unix week bucket from STRATZ
	MatchCount int    `json:"matchCount"`
	WinCount   int    `json:"winCount"`
}

// BracketResponse is STRATZ's reply for one bracket.
type BracketResponse struct {
	Bracket Bracket
	Weeks   []HeroWeekStat // flattened across all heroes
}
```

- [ ] **Step 2: Commit the types file**

```bash
git add internal/api/stratz/types.go
git commit -m "feat(stratz): add bracket + hero week stat types"
```

- [ ] **Step 3: Write `client.go` — Fetch and Parse separated for testability**

```go
package stratz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const endpoint = "https://api.stratz.com/graphql"

// Client talks to STRATZ GraphQL.
type Client struct {
	Token      string
	HTTP       *http.Client
	Endpoint   string // defaults to stratz production; overridable in tests
}

// NewClient returns a Client with sane defaults.
func NewClient(token string) *Client {
	return &Client{
		Token:    token,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
		Endpoint: endpoint,
	}
}

const bracketQuery = `query HeroWeekStats($bracket: RankBracket!) {
  heroStats {
    winWeek(bracketIds: [$bracket], take: 20) {
      heroId
      week
      matchCount
      winCount
    }
  }
}`

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data struct {
		HeroStats struct {
			WinWeek []HeroWeekStat `json:"winWeek"`
		} `json:"heroStats"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchBracket returns 20 weekly entries per hero for one bracket.
func (c *Client) FetchBracket(bracket Bracket) (BracketResponse, error) {
	body, err := json.Marshal(graphQLRequest{
		Query:     bracketQuery,
		Variables: map[string]any{"bracket": string(bracket)},
	})
	if err != nil {
		return BracketResponse{}, fmt.Errorf("encoding stratz request: %w", err)
	}

	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return BracketResponse{}, fmt.Errorf("building stratz request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("User-Agent", "dota-meta/1.0")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return BracketResponse{}, fmt.Errorf("stratz request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return BracketResponse{}, fmt.Errorf("stratz returned %d: %s", resp.StatusCode, string(snippet))
	}

	return parseBracket(bracket, resp.Body)
}

func parseBracket(bracket Bracket, r io.Reader) (BracketResponse, error) {
	var raw graphQLResponse
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return BracketResponse{}, fmt.Errorf("decoding stratz response: %w", err)
	}
	if len(raw.Errors) > 0 {
		return BracketResponse{}, fmt.Errorf("stratz graphql error: %s", raw.Errors[0].Message)
	}
	return BracketResponse{Bracket: bracket, Weeks: raw.Data.HeroStats.WinWeek}, nil
}
```

- [ ] **Step 4: Capture a small test fixture**

Hand-author `internal/api/stratz/testdata/response.json` with two hero rows × two weeks. Real structure, compact:

```json
{
  "data": {
    "heroStats": {
      "winWeek": [
        {"heroId": 1, "week": 2850, "matchCount": 12000, "winCount": 6300},
        {"heroId": 1, "week": 2849, "matchCount": 11500, "winCount": 5900},
        {"heroId": 2, "week": 2850, "matchCount": 8000, "winCount": 4100},
        {"heroId": 2, "week": 2849, "matchCount": 7800, "winCount": 3800}
      ]
    }
  }
}
```

- [ ] **Step 5: Write `client_test.go`**

```go
package stratz

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetchBracket_ParsesFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/response.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testtoken" {
			t.Errorf("missing/wrong bearer: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	c := NewClient("testtoken")
	c.Endpoint = srv.URL

	got, err := c.FetchBracket(BracketDivine)
	if err != nil {
		t.Fatalf("FetchBracket: %v", err)
	}

	if got.Bracket != BracketDivine {
		t.Errorf("bracket = %q, want DIVINE", got.Bracket)
	}
	if len(got.Weeks) != 4 {
		t.Fatalf("len(Weeks)=%d, want 4", len(got.Weeks))
	}
	if got.Weeks[0].HeroID != 1 || got.Weeks[0].MatchCount != 12000 {
		t.Errorf("first row mismatch: %+v", got.Weeks[0])
	}
}

func TestFetchBracket_GraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"errors":[{"message":"invalid bracket"}]}`))
	}))
	defer srv.Close()

	c := NewClient("x")
	c.Endpoint = srv.URL

	_, err := c.FetchBracket(BracketDivine)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 6: Run tests and vet**

```bash
go test ./internal/api/stratz/... -v
go vet ./...
```
Expected: both tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/api/stratz/
git commit -m "feat(stratz): graphql client with bracket winWeek fetch"
```

---

## Task 3 — STRATZ: fetch all brackets + hero metadata, adapt to internal `Hero` shape

STRATZ `winWeek` gives per-bracket stats keyed by `heroId` — no hero names/roles. We still need hero metadata. STRATZ exposes `constants.heroes` which returns id, `displayName`, `roles`, `primaryAttribute`, `attackType`, `shortName` (for img). Extend the client to fetch those once per run and merge.

**Files:**
- Modify: `internal/api/stratz/client.go`, `types.go`, `client_test.go`.

- [ ] **Step 1: Extend `types.go` with `Hero` metadata struct**

```go
// Hero is the bracket-agnostic hero catalog entry (from constants.heroes).
type Hero struct {
	ID               int      `json:"id"`
	ShortName        string   `json:"shortName"`        // "anti-mage" for img URL
	DisplayName      string   `json:"displayName"`
	Roles            []string `json:"roles"`
	PrimaryAttribute string   `json:"primaryAttribute"` // "STR"|"AGI"|"INT"|"ALL"
	AttackType       string   `json:"attackType"`       // "Melee"|"Ranged"
}
```

- [ ] **Step 2: Extend `client.go` with `FetchHeroes`**

```go
const heroesQuery = `query Heroes {
  constants {
    heroes {
      id
      shortName
      displayName
      roles { roleId }
      stats { primaryAttributeEnum attackType }
    }
  }
}`

// FetchHeroes returns the hero catalog. Call once per run.
func (c *Client) FetchHeroes() ([]Hero, error) {
	body, _ := json.Marshal(graphQLRequest{Query: heroesQuery})
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building heroes request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("heroes request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("stratz heroes returned %d: %s", resp.StatusCode, string(snippet))
	}
	return parseHeroes(resp.Body)
}
```

**Note:** STRATZ `roles` is `[{roleId, level}]` where `roleId` is an int. For v1, don't try to map to strings — just record the raw role ids and add a helper. But the analysis only cares whether a hero is Support and whether it's also Carry. STRATZ role ids: `CARRY=1, SUPPORT=6` per their schema. Keep it local to the package.

Add a `parseHeroes` that decodes the raw shape and flattens to `Hero.Roles` as a `[]string{"Carry","Support",...}` using this map:

```go
var roleIDToName = map[int]string{
	1: "Carry", 2: "Disabler", 3: "Durable", 4: "Escape",
	5: "Initiator", 6: "Support", 7: "Nuker", 8: "Pusher", 9: "Jungler",
}

func parseHeroes(r io.Reader) ([]Hero, error) {
	var raw struct {
		Data struct {
			Constants struct {
				Heroes []struct {
					ID          int    `json:"id"`
					ShortName   string `json:"shortName"`
					DisplayName string `json:"displayName"`
					Roles       []struct {
						RoleID int `json:"roleId"`
					} `json:"roles"`
					Stats struct {
						PrimaryAttributeEnum string `json:"primaryAttributeEnum"`
						AttackType           string `json:"attackType"`
					} `json:"stats"`
				} `json:"heroes"`
			} `json:"constants"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding heroes: %w", err)
	}
	out := make([]Hero, 0, len(raw.Data.Constants.Heroes))
	for _, h := range raw.Data.Constants.Heroes {
		var roles []string
		for _, r := range h.Roles {
			if name, ok := roleIDToName[r.RoleID]; ok {
				roles = append(roles, name)
			}
		}
		out = append(out, Hero{
			ID: h.ID, ShortName: h.ShortName, DisplayName: h.DisplayName,
			Roles: roles, PrimaryAttribute: h.Stats.PrimaryAttributeEnum,
			AttackType: h.Stats.AttackType,
		})
	}
	return out, nil
}
```

- [ ] **Step 3: Add fixture `testdata/heroes.json`**

Two-hero sample matching the shape above.

```json
{
  "data": {
    "constants": {
      "heroes": [
        {"id": 1, "shortName": "antimage", "displayName": "Anti-Mage",
         "roles": [{"roleId": 1}, {"roleId": 4}],
         "stats": {"primaryAttributeEnum": "AGI", "attackType": "Melee"}},
        {"id": 2, "shortName": "axe", "displayName": "Axe",
         "roles": [{"roleId": 5}, {"roleId": 3}],
         "stats": {"primaryAttributeEnum": "STR", "attackType": "Melee"}}
      ]
    }
  }
}
```

- [ ] **Step 4: Extend `client_test.go`**

```go
func TestFetchHeroes_ParsesFixture(t *testing.T) {
	fixture, _ := os.ReadFile("testdata/heroes.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()
	c := NewClient("x"); c.Endpoint = srv.URL
	got, err := c.FetchHeroes()
	if err != nil { t.Fatalf("FetchHeroes: %v", err) }
	if len(got) != 2 { t.Fatalf("len=%d, want 2", len(got)) }
	if got[0].DisplayName != "Anti-Mage" { t.Errorf("bad name: %s", got[0].DisplayName) }
	if !contains(got[0].Roles, "Carry") { t.Errorf("missing Carry: %v", got[0].Roles) }
}

func contains(s []string, v string) bool {
	for _, x := range s { if x == v { return true } }
	return false
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/api/stratz/... -v
```
Expected: 3 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/stratz/
git commit -m "feat(stratz): fetch hero constants and map role ids to names"
```

---

## Task 4 — STRATZ: `FetchAll` aggregating brackets with rate limiting

**Files:**
- Modify: `internal/api/stratz/client.go`, `client_test.go`.

- [ ] **Step 1: Add `FetchAll`**

```go
// FetchAll fetches hero metadata plus 8 bracket histories sequentially with a
// small sleep to stay well under the 7 rps rate limit.
func (c *Client) FetchAll() ([]Hero, []BracketResponse, error) {
	heroes, err := c.FetchHeroes()
	if err != nil {
		return nil, nil, err
	}
	brackets := make([]BracketResponse, 0, len(AllBrackets()))
	for i, b := range AllBrackets() {
		if i > 0 {
			time.Sleep(200 * time.Millisecond) // ~5 rps ceiling
		}
		br, err := c.FetchBracket(b)
		if err != nil {
			return nil, nil, fmt.Errorf("bracket %s: %w", b, err)
		}
		brackets = append(brackets, br)
	}
	return heroes, brackets, nil
}
```

- [ ] **Step 2: Add a unit test using the test server to count request count + Authorization**

```go
func TestFetchAll_MakesNinePlusRequests(t *testing.T) {
	heroesFixture, _ := os.ReadFile("testdata/heroes.json")
	bracketFixture, _ := os.ReadFile("testdata/response.json")
	var count int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		body, _ := io.ReadAll(r.Body)
		if bytes.Contains(body, []byte("constants")) {
			w.Write(heroesFixture)
			return
		}
		w.Write(bracketFixture)
	}))
	defer srv.Close()
	c := NewClient("x"); c.Endpoint = srv.URL
	heroes, brackets, err := c.FetchAll()
	if err != nil { t.Fatalf("FetchAll: %v", err) }
	if len(heroes) == 0 { t.Error("no heroes") }
	if len(brackets) != 8 { t.Errorf("brackets=%d, want 8", len(brackets)) }
	if count != 9 { t.Errorf("http calls=%d, want 9", count) }
}
```

- [ ] **Step 3: Run**

```bash
go test ./internal/api/stratz/... -v
```
Expected: 4 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/api/stratz/
git commit -m "feat(stratz): FetchAll aggregates heroes and 8 bracket responses"
```

---

## Task 5 — Analysis: bridge STRATZ data into internal `HeroStat` shape

STRATZ gives per-hero weekly counts per bracket. Analysis needs aggregate pick/win per hero per "analysis bracket" plus the weekly series. Because STRATZ brackets are a finer 8-split and spec calls out Immortal as its own bucket, the analysis brackets become:

```
Herald-Guardian   = [HERALD, GUARDIAN]
Crusader-Archon   = [CRUSADER, ARCHON]
Legend-Ancient    = [LEGEND, ANCIENT]
Divine            = [DIVINE]
Immortal          = [IMMORTAL]
```

**Files:**
- Create: `internal/api/stratz/bridge.go` — pure function to roll weekly rows up.
- Create: `internal/api/stratz/bridge_test.go`.

- [ ] **Step 1: Write the bridge function**

```go
package stratz

// AggregatedHero is one hero's totals + weekly series inside an analysis bracket.
type AggregatedHero struct {
	HeroID    int
	Picks     int
	Wins      int
	WeeklyWR  []float64 // len == 20 if STRATZ returns 20 weeks; newest last
}

// AggregateBrackets rolls up a list of STRATZ bracket responses (each with one
// bracket's weekly rows) into one AggregatedHero per (heroID). Callers pass
// only the brackets that belong to a single analysis bucket.
func AggregateBrackets(responses []BracketResponse) map[int]AggregatedHero {
	// Step 1: sum picks/wins per hero across all inputs.
	// Step 2: build weekly series per hero by summing matchCount/winCount per
	//         week bucket across input brackets, then emit WR oldest→newest.
	type weekAgg struct{ picks, wins int }
	type acc struct {
		totalPicks, totalWins int
		weekly               map[int64]weekAgg
	}
	byHero := map[int]*acc{}
	for _, br := range responses {
		for _, w := range br.Weeks {
			a, ok := byHero[w.HeroID]
			if !ok {
				a = &acc{weekly: map[int64]weekAgg{}}
				byHero[w.HeroID] = a
			}
			a.totalPicks += w.MatchCount
			a.totalWins += w.WinCount
			wa := a.weekly[w.Week]
			wa.picks += w.MatchCount
			wa.wins += w.WinCount
			a.weekly[w.Week] = wa
		}
	}
	out := make(map[int]AggregatedHero, len(byHero))
	for id, a := range byHero {
		weeks := make([]int64, 0, len(a.weekly))
		for w := range a.weekly { weeks = append(weeks, w) }
		sort.Slice(weeks, func(i, j int) bool { return weeks[i] < weeks[j] })
		series := make([]float64, 0, len(weeks))
		for _, w := range weeks {
			x := a.weekly[w]
			if x.picks == 0 { series = append(series, 0); continue }
			series = append(series, float64(x.wins)/float64(x.picks)*100)
		}
		out[id] = AggregatedHero{
			HeroID: id, Picks: a.totalPicks, Wins: a.totalWins, WeeklyWR: series,
		}
	}
	return out
}
```

Add `"sort"` to imports.

- [ ] **Step 2: Write tests covering single-bracket and 2-bracket aggregation**

```go
package stratz

import (
	"math"
	"testing"
)

func TestAggregateBrackets_SingleBracket(t *testing.T) {
	resp := []BracketResponse{{
		Bracket: BracketDivine,
		Weeks: []HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 55},
			{HeroID: 1, Week: 101, MatchCount: 200, WinCount: 110},
		},
	}}
	got := AggregateBrackets(resp)
	h := got[1]
	if h.Picks != 300 || h.Wins != 165 {
		t.Errorf("totals=%d/%d, want 300/165", h.Picks, h.Wins)
	}
	if len(h.WeeklyWR) != 2 { t.Fatalf("weekly len=%d, want 2", len(h.WeeklyWR)) }
	// oldest first: week 100 -> 55%, week 101 -> 55%
	if math.Abs(h.WeeklyWR[0]-55.0) > 1e-6 { t.Errorf("wr0=%f", h.WeeklyWR[0]) }
}

func TestAggregateBrackets_MergesTwoBrackets(t *testing.T) {
	a := BracketResponse{Bracket: BracketHerald, Weeks: []HeroWeekStat{
		{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 50},
	}}
	b := BracketResponse{Bracket: BracketGuardian, Weeks: []HeroWeekStat{
		{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 60},
	}}
	got := AggregateBrackets([]BracketResponse{a, b})
	if got[1].Picks != 200 || got[1].Wins != 110 {
		t.Errorf("merge totals=%d/%d, want 200/110", got[1].Picks, got[1].Wins)
	}
	if math.Abs(got[1].WeeklyWR[0]-55.0) > 1e-6 {
		t.Errorf("merged wr=%f, want 55.0", got[1].WeeklyWR[0])
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/api/stratz/... -v
```
Expected: 6 tests PASS total.

- [ ] **Step 4: Commit**

```bash
git add internal/api/stratz/
git commit -m "feat(stratz): aggregate weekly rows into analysis buckets"
```

---

## Task 6 — Redefine analysis `Brackets` to include Immortal

**Files:**
- Modify: `internal/analysis/analysis.go` — `Brackets` variable and `Bracket` struct semantics.

Goal: decouple `Bracket.Indices` (OpenDota 0-indexed ints) from the STRATZ bracket set. Replace with string IDs that both sources can produce, or add a parallel STRATZ-native bracket list. **Simpler path:** keep `Bracket.Indices` for OpenDota only, and add a second const list `StratzBrackets` that references `stratz.Bracket`. Callers pick which to use.

Actually simplest: since we're switching primary to STRATZ, redefine `Brackets` as the new 5-entry list and delete the OpenDota bracket coupling from this file. OpenDota bracket logic stays inside the archived package only.

- [ ] **Step 1: Rewrite `Bracket` struct and `Brackets` list**

Replace the current `Bracket` struct and `Brackets` var with:

```go
// Bracket is an analysis bucket — one or more STRATZ ranks grouped together.
type Bracket struct {
	Name    string
	Stratz  []stratz.Bracket
}

// Brackets defines the 5 analysis buckets. Immortal is now its own bucket
// (STRATZ separates it from Divine, unlike OpenDota).
var Brackets = []Bracket{
	{Name: "Herald-Guardian", Stratz: []stratz.Bracket{stratz.BracketHerald, stratz.BracketGuardian}},
	{Name: "Crusader-Archon", Stratz: []stratz.Bracket{stratz.BracketCrusader, stratz.BracketArchon}},
	{Name: "Legend-Ancient",  Stratz: []stratz.Bracket{stratz.BracketLegend, stratz.BracketAncient}},
	{Name: "Divine",          Stratz: []stratz.Bracket{stratz.BracketDivine}},
	{Name: "Immortal",        Stratz: []stratz.Bracket{stratz.BracketImmortal}},
}
```

Add import `"github.com/narrowcastdev/dota-meta/internal/api/stratz"`.

- [ ] **Step 2: Remove stale OpenDota wiring from `analysis.go`**

Delete `sumIndices`, `analyzeBracket` (old OpenDota version), `analyzeBracketDelta`, `Analyze`. Delete the `opendota` import. Keep the struct definitions of `HeroStat`, `BracketAnalysis`, `FullAnalysis`, `DeltaHero` — they're source-agnostic. They'll be repopulated by the new STRATZ-driven `Analyze` in Task 7.

Change `HeroStat.Hero` field type from `opendota.Hero` to `stratz.Hero`.

- [ ] **Step 3: Update `analysis_test.go`**

The existing tests reference OpenDota types. Either gut them now (they'll be replaced in Task 7) or move them under a build tag. Simpler: delete `internal/analysis/analysis_test.go`. Task 7 and later tasks add new tests.

```bash
git rm internal/analysis/analysis_test.go
```

- [ ] **Step 4: Compile-only sanity check**

```bash
go build ./...
```
Expected: build errors *only* in `format/` and `cmd/` (they use deleted funcs). That's fine — Task 7 and 13 patch those callers. Do not try to fix them yet.

- [ ] **Step 5: Commit (build intentionally broken at format/cmd; core package compiles)**

Run `go build ./internal/analysis/... ./internal/api/...` to confirm those at least build.

```bash
go build ./internal/analysis/... ./internal/api/...
git add internal/analysis/
git commit -m "refactor(analysis): redefine brackets around STRATZ, drop OpenDota coupling"
```

*(Yes, the repo does not build end-to-end between this task and Task 7. Acceptable because they run back-to-back and the executor reviews after each task. Call it out in the PR description if shipping.)*

---

## Task 7 — Analysis: new `Analyze` over STRATZ data (no tiers yet)

**Files:**
- Modify: `internal/analysis/analysis.go`.
- Create: `internal/analysis/analyze_test.go`.

- [ ] **Step 1: Write new `Analyze` and `analyzeBracket`**

```go
// Analyze builds a FullAnalysis from STRATZ hero catalog + per-STRATZ-bracket
// weekly responses. `responses` must contain one entry per STRATZ bracket we
// care about. Missing brackets → empty buckets (no panic).
func Analyze(heroes []stratz.Hero, responses []stratz.BracketResponse, minPicks int) FullAnalysis {
	byID := make(map[int]stratz.Hero, len(heroes))
	for _, h := range heroes { byID[h.ID] = h }

	respByBracket := make(map[stratz.Bracket]stratz.BracketResponse, len(responses))
	for _, r := range responses { respByBracket[r.Bracket] = r }

	var full FullAnalysis
	for _, b := range Brackets {
		bucketResps := make([]stratz.BracketResponse, 0, len(b.Stratz))
		for _, sb := range b.Stratz {
			if r, ok := respByBracket[sb]; ok { bucketResps = append(bucketResps, r) }
		}
		agg := stratz.AggregateBrackets(bucketResps)
		ba := analyzeBracket(b, byID, agg, minPicks)
		full.Brackets = append(full.Brackets, ba)
		full.TotalMatches += ba.Matches()
	}
	return full
}

func analyzeBracket(bracket Bracket, byID map[int]stratz.Hero, agg map[int]stratz.AggregatedHero, minPicks int) BracketAnalysis {
	var totalPicks int
	for _, h := range agg { totalPicks += h.Picks }
	matches := totalPicks / picksPerMatch
	ba := BracketAnalysis{Bracket: bracket, TotalPicks: totalPicks}
	for id, a := range agg {
		if a.Picks < minPicks { continue }
		hero, ok := byID[id]
		if !ok { continue } // unknown hero id, skip
		wr := 0.0
		if a.Picks > 0 { wr = float64(a.Wins) / float64(a.Picks) * 100 }
		pr := 0.0
		if matches > 0 { pr = float64(a.Picks) / float64(matches) * 100 }
		hs := HeroStat{
			Hero:      hero,
			Picks:     a.Picks,
			Wins:      a.Wins,
			WinRate:   wr,
			PickRate:  pr,
			WRHistory: a.WeeklyWR,
		}
		if isSupport(hero) {
			ba.Supports = append(ba.Supports, hs)
		} else {
			ba.Cores = append(ba.Cores, hs)
		}
	}
	return ba
}

func isSupport(h stratz.Hero) bool {
	var support, carry bool
	for _, r := range h.Roles {
		if r == "Support" { support = true }
		if r == "Carry" { carry = true }
	}
	return support && !carry
}
```

Update `HeroStat` to include `WRHistory []float64` if not already. Remove old `Best/Sleepers/Traps` from `BracketAnalysis`; add `Cores []HeroStat` and `Supports []HeroStat`. Keep `Matches()`.

`FullAnalysis` gets new fields now (most set later):

```go
type FullAnalysis struct {
	Brackets         []BracketAnalysis
	TotalMatches     int
	Patch            string
	SnapshotDate     time.Time
	PriorSnapshot    *time.Time
	SnapshotsInPatch int
	// delta/climb fields live inside HeroStat, not here
}
```

Drop `LowStompers`, `HighSkillCap` — climb tagging replaces that (Task 9).

- [ ] **Step 2: Write table tests**

```go
package analysis

import (
	"testing"

	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

func TestAnalyze_SplitsCoresAndSupports(t *testing.T) {
	heroes := []stratz.Hero{
		{ID: 1, DisplayName: "Anti-Mage", Roles: []string{"Carry"}},
		{ID: 2, DisplayName: "Crystal Maiden", Roles: []string{"Support", "Disabler"}},
		{ID: 3, DisplayName: "Wraith King", Roles: []string{"Carry", "Support"}}, // flex → core
	}
	// single bracket: DIVINE with enough picks.
	resp := []stratz.BracketResponse{{
		Bracket: stratz.BracketDivine,
		Weeks: []stratz.HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 2000, WinCount: 1100},
			{HeroID: 2, Week: 100, MatchCount: 2000, WinCount: 1050},
			{HeroID: 3, Week: 100, MatchCount: 2000, WinCount: 900},
		},
	}}
	got := Analyze(heroes, resp, 1000)
	divine := findBracket(t, got, "Divine")
	coreNames := names(divine.Cores)
	suppNames := names(divine.Supports)
	if !hasName(coreNames, "Anti-Mage") { t.Errorf("AM should be core: %v", coreNames) }
	if !hasName(coreNames, "Wraith King") { t.Errorf("WK (flex) should be core: %v", coreNames) }
	if !hasName(suppNames, "Crystal Maiden") { t.Errorf("CM should be support: %v", suppNames) }
}

func TestAnalyze_HonorsMinPicks(t *testing.T) {
	heroes := []stratz.Hero{{ID: 1, DisplayName: "Anti-Mage", Roles: []string{"Carry"}}}
	resp := []stratz.BracketResponse{{
		Bracket: stratz.BracketDivine,
		Weeks: []stratz.HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 500, WinCount: 250},
		},
	}}
	got := Analyze(heroes, resp, 1000)
	divine := findBracket(t, got, "Divine")
	if len(divine.Cores)+len(divine.Supports) != 0 {
		t.Errorf("expected 0 qualified, got %d cores %d supports", len(divine.Cores), len(divine.Supports))
	}
}

func findBracket(t *testing.T, a FullAnalysis, name string) BracketAnalysis {
	t.Helper()
	for _, b := range a.Brackets { if b.Bracket.Name == name { return b } }
	t.Fatalf("bracket %q not found", name); return BracketAnalysis{}
}
func names(stats []HeroStat) []string {
	out := make([]string, 0, len(stats))
	for _, s := range stats { out = append(out, s.Hero.DisplayName) }
	return out
}
func hasName(xs []string, v string) bool { for _, x := range xs { if x == v { return true } }; return false }
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/analysis/... -v
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/analysis/
git commit -m "feat(analysis): Analyze over stratz aggregates, cores/supports split"
```

---

## Task 8 — Tier quadrants (`ClassifyTiers`)

**Files:**
- Create: `internal/analysis/tier.go`.
- Create: `internal/analysis/tier_test.go`.
- Modify: `internal/analysis/analysis.go` — call `ClassifyTiers` at end of `analyzeBracket`.

- [ ] **Step 1: Write failing test first**

`internal/analysis/tier_test.go`:

```go
package analysis

import "testing"

func TestClassifyTiers_MedianSplit(t *testing.T) {
	// 4 heroes: (WR, PR) pairs designed so medians split cleanly.
	stats := []HeroStat{
		{WinRate: 60, PickRate: 10},  // high WR, high PR → Meta Tyrant
		{WinRate: 58, PickRate: 2},   // high WR, low PR → Pocket Pick
		{WinRate: 45, PickRate: 12},  // low WR, high PR → Trap
		{WinRate: 43, PickRate: 1},   // low WR, low PR → Dead
	}
	ClassifyTiers(stats)
	want := map[int]Tier{0: TierMetaTyrant, 1: TierPocketPick, 2: TierTrap, 3: TierDead}
	for i, w := range want {
		if stats[i].Tier != w {
			t.Errorf("stats[%d].Tier=%v, want %v", i, stats[i].Tier, w)
		}
	}
}

func TestClassifyTiers_Empty(t *testing.T) {
	var stats []HeroStat
	ClassifyTiers(stats) // must not panic
}

func TestClassifyTiers_SingleHero(t *testing.T) {
	stats := []HeroStat{{WinRate: 50, PickRate: 5}}
	ClassifyTiers(stats)
	// With n=1, median == the only value → hero is at both medians → MetaTyrant.
	if stats[0].Tier != TierMetaTyrant {
		t.Errorf("got %v, want Meta Tyrant", stats[0].Tier)
	}
}
```

- [ ] **Step 2: Run — should fail to compile**

```bash
go test ./internal/analysis/... -run TestClassifyTiers -v
```
Expected: FAIL — `ClassifyTiers` undefined.

- [ ] **Step 3: Implement `tier.go`**

```go
package analysis

import "sort"

// Tier is one of four quadrants derived from WR vs PR medians.
type Tier int

const (
	TierDead Tier = iota
	TierTrap
	TierPocketPick
	TierMetaTyrant
)

func (t Tier) String() string {
	switch t {
	case TierMetaTyrant: return "meta-tyrant"
	case TierPocketPick: return "pocket-pick"
	case TierTrap:       return "trap"
	default:             return "dead"
	}
}

// ClassifyTiers mutates stats in place, assigning Tier based on within-slice
// median WR and median PR splits. High-or-equal to median counts as "high".
func ClassifyTiers(stats []HeroStat) {
	if len(stats) == 0 { return }
	wr := make([]float64, len(stats))
	pr := make([]float64, len(stats))
	for i, s := range stats {
		wr[i] = s.WinRate
		pr[i] = s.PickRate
	}
	sort.Float64s(wr)
	sort.Float64s(pr)
	medWR := median(wr)
	medPR := median(pr)
	for i := range stats {
		highWR := stats[i].WinRate >= medWR
		highPR := stats[i].PickRate >= medPR
		switch {
		case highWR && highPR:  stats[i].Tier = TierMetaTyrant
		case highWR && !highPR: stats[i].Tier = TierPocketPick
		case !highWR && highPR: stats[i].Tier = TierTrap
		default:                stats[i].Tier = TierDead
		}
	}
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 { return 0 }
	if n%2 == 1 { return sorted[n/2] }
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
```

Add `Tier Tier` field to `HeroStat` in `analysis.go`.

- [ ] **Step 4: Run tier tests — should PASS**

```bash
go test ./internal/analysis/... -run TestClassifyTiers -v
```

- [ ] **Step 5: Wire into `analyzeBracket`**

At the end of `analyzeBracket`, just before returning:

```go
ClassifyTiers(ba.Cores)
ClassifyTiers(ba.Supports)
```

**Separate medians per role family** per spec default.

- [ ] **Step 6: Run the full analysis test suite**

```bash
go test ./internal/analysis/... -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/analysis/
git commit -m "feat(analysis): 2x2 tier quadrants per bracket, per role family"
```

---

## Task 9 — Bracket-climb tag

**Files:**
- Create: `internal/analysis/climb.go`
- Create: `internal/analysis/climb_test.go`
- Modify: `internal/analysis/analysis.go` — compute and attach climb tag on each `HeroStat` *in every bracket*.

Spec: delta = DivineWR − HeraldWR. Threshold ±2.0pp. Only tag if hero qualifies in both Herald-Guardian and Divine.

- [ ] **Step 1: Test**

```go
package analysis

import "testing"

func TestTagClimb(t *testing.T) {
	cases := []struct {
		lowWR, highWR float64
		qualified     bool
		want          ClimbTag
	}{
		{lowWR: 50, highWR: 53, qualified: true, want: ClimbUp},
		{lowWR: 53, highWR: 50, qualified: true, want: ClimbDown},
		{lowWR: 51, highWR: 52, qualified: true, want: ClimbUniversal},
		{lowWR: 51, highWR: 52, qualified: false, want: ClimbUnknown},
	}
	for _, c := range cases {
		got := TagClimb(c.lowWR, c.highWR, c.qualified)
		if got != c.want {
			t.Errorf("TagClimb(%v,%v,%v)=%v, want %v", c.lowWR, c.highWR, c.qualified, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Implement**

```go
package analysis

type ClimbTag string

const (
	ClimbUnknown   ClimbTag = ""
	ClimbUp        ClimbTag = "scales-up"
	ClimbDown      ClimbTag = "scales-down"
	ClimbUniversal ClimbTag = "universal"
)

const climbDeltaThreshold = 2.0

// TagClimb returns the climb tag for a hero given its Herald-Guardian WR and
// Divine WR. If the hero didn't meet the min-picks floor in one of the brackets,
// returns ClimbUnknown.
func TagClimb(lowWR, highWR float64, qualified bool) ClimbTag {
	if !qualified { return ClimbUnknown }
	delta := highWR - lowWR
	switch {
	case delta >= climbDeltaThreshold:  return ClimbUp
	case delta <= -climbDeltaThreshold: return ClimbDown
	default:                            return ClimbUniversal
	}
}
```

Add `ClimbTag ClimbTag` field to `HeroStat`.

- [ ] **Step 3: Wire into `Analyze`**

After all brackets are analyzed, build a hero-id → (lowWR, highWR, qualified-in-both) map from `Herald-Guardian` and `Divine` buckets, then walk every `HeroStat` in every bracket and set its `ClimbTag`.

```go
// inside Analyze, after the for-bracket loop:
type wrPair struct{ low, high float64; hasLow, hasHigh bool }
refs := map[int]wrPair{}
for _, ba := range full.Brackets {
	for _, section := range [][]HeroStat{ba.Cores, ba.Supports} {
		for _, s := range section {
			p := refs[s.Hero.ID]
			if ba.Bracket.Name == "Herald-Guardian" { p.low = s.WinRate; p.hasLow = true }
			if ba.Bracket.Name == "Divine"          { p.high = s.WinRate; p.hasHigh = true }
			refs[s.Hero.ID] = p
		}
	}
}
for bi := range full.Brackets {
	applyClimb := func(stats []HeroStat) {
		for si := range stats {
			p := refs[stats[si].Hero.ID]
			stats[si].ClimbTag = TagClimb(p.low, p.high, p.hasLow && p.hasHigh)
		}
	}
	applyClimb(full.Brackets[bi].Cores)
	applyClimb(full.Brackets[bi].Supports)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/analysis/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/analysis/
git commit -m "feat(analysis): climb tag per hero using Herald vs Divine delta"
```

---

## Task 10 — Snapshot history: patch field, save/load

**Files:**
- Create: `internal/analysis/history.go`.
- Create: `internal/analysis/history_test.go`.
- Modify: `update.sh` — copy before regenerate.
- Modify: `cmd/dota-meta/main.go` — write patch + snapshotDate into output.
- Create directory: `docs/history/` with a `.gitkeep`.

Stored snapshot is the raw `docs/data.json`. On each run we try to load the most recent `docs/history/data-YYYY-MM-DD.json` (strictly older than today) and compute deltas.

- [ ] **Step 1: Write `history.go`**

```go
package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// snapshotSummary is the minimal shape we need from a prior snapshot —
// (bracket, heroName) → WR/PR. Uses display-name key for stability across runs.
type snapshotSummary struct {
	Date     time.Time
	Patch    string
	Stats    map[string]map[string]struct{ WR, PR float64 } // bracketName → heroName → stats
}

// LoadLatestPriorSnapshot returns the newest snapshot in dir strictly older
// than today. Returns nil, nil if none exist.
func LoadLatestPriorSnapshot(dir string, today time.Time) (*snapshotSummary, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) { return nil, nil }
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	type dated struct{ date time.Time; path string }
	var candidates []dated
	for _, e := range entries {
		if e.IsDir() { continue }
		name := e.Name()
		if !strings.HasPrefix(name, "data-") || !strings.HasSuffix(name, ".json") { continue }
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "data-"), ".json")
		d, err := time.Parse("2006-01-02", datePart)
		if err != nil { continue }
		if !d.Before(today) { continue }
		candidates = append(candidates, dated{d, filepath.Join(dir, name)})
	}
	if len(candidates) == 0 { return nil, nil }
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].date.After(candidates[j].date) })
	newest := candidates[0]
	data, err := os.ReadFile(newest.path)
	if err != nil { return nil, err }
	return parseSnapshot(newest.date, data)
}

// parseSnapshot reads the format written by format.FormatJSON.
// Only Patch, and per-bracket (heroName → WR/PR) are extracted.
func parseSnapshot(date time.Time, data []byte) (*snapshotSummary, error) {
	var shape struct {
		Patch    string `json:"patch"`
		Analysis struct {
			Brackets map[string]struct {
				Name  string `json:"name"`  // human-readable "Herald-Guardian" etc.
				Cores []struct {
					Name     string  `json:"name"`
					WinRate  float64 `json:"win_rate"`
					PickRate float64 `json:"pick_rate"`
				} `json:"cores"`
				Supports []struct {
					Name     string  `json:"name"`
					WinRate  float64 `json:"win_rate"`
					PickRate float64 `json:"pick_rate"`
				} `json:"supports"`
			} `json:"brackets"`
		} `json:"analysis"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}
	out := &snapshotSummary{Date: date, Patch: shape.Patch, Stats: map[string]map[string]struct{WR, PR float64}{}}
	for _, b := range shape.Analysis.Brackets {
		m := map[string]struct{WR, PR float64}{}
		for _, s := range b.Cores    { m[s.Name] = struct{WR, PR float64}{s.WinRate, s.PickRate} }
		for _, s := range b.Supports { m[s.Name] = struct{WR, PR float64}{s.WinRate, s.PickRate} }
		out.Stats[b.Name] = m
	}
	return out, nil
}

// ApplyDeltas fills WRDelta/PRDelta on every HeroStat in the analysis using
// the given prior snapshot. Does nothing (leaves nils) if prior is nil.
func ApplyDeltas(full *FullAnalysis, prior *snapshotSummary) {
	if prior == nil { return }
	full.PriorSnapshot = &prior.Date
	for bi := range full.Brackets {
		apply := func(stats []HeroStat, bracketName string) {
			bracketMap, ok := prior.Stats[bracketName]
			if !ok { return }
			for si := range stats {
				old, ok := bracketMap[stats[si].Hero.DisplayName]
				if !ok { continue }
				wrD := stats[si].WinRate - old.WR
				prD := stats[si].PickRate - old.PR
				stats[si].WRDelta = &wrD
				stats[si].PRDelta = &prD
			}
		}
		apply(full.Brackets[bi].Cores,    full.Brackets[bi].Bracket.Name)
		apply(full.Brackets[bi].Supports, full.Brackets[bi].Bracket.Name)
	}
}
```

Add new fields on `HeroStat`:

```go
WRDelta *float64
PRDelta *float64
```

- [ ] **Step 2: Write tests**

`internal/analysis/history_test.go`:

```go
package analysis

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

func writeSnapshot(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLatestPriorSnapshot_PicksNewestBeforeToday(t *testing.T) {
	dir := t.TempDir()
	writeSnapshot(t, dir, "data-2026-04-10.json", `{"patch":"7.40a","analysis":{"brackets":{"d":{"name":"Divine","cores":[{"name":"AM","win_rate":50,"pick_rate":5}],"supports":[]}}}}`)
	writeSnapshot(t, dir, "data-2026-04-15.json", `{"patch":"7.40b","analysis":{"brackets":{"d":{"name":"Divine","cores":[{"name":"AM","win_rate":52,"pick_rate":6}],"supports":[]}}}}`)
	writeSnapshot(t, dir, "data-2026-04-23.json", `{"patch":"7.40b","analysis":{"brackets":{"d":{"name":"Divine","cores":[{"name":"AM","win_rate":99,"pick_rate":9}],"supports":[]}}}}`) // == today, ignored
	today, _ := time.Parse("2006-01-02", "2026-04-23")
	got, err := LoadLatestPriorSnapshot(dir, today)
	if err != nil { t.Fatal(err) }
	if got == nil { t.Fatal("got nil") }
	if got.Patch != "7.40b" { t.Errorf("patch=%q want 7.40b", got.Patch) }
	if got.Stats["Divine"]["AM"].WR != 52 { t.Errorf("wr=%v want 52", got.Stats["Divine"]["AM"].WR) }
}

func TestLoadLatestPriorSnapshot_NoDir(t *testing.T) {
	today := time.Now()
	got, err := LoadLatestPriorSnapshot(filepath.Join(t.TempDir(), "nope"), today)
	if err != nil { t.Fatal(err) }
	if got != nil { t.Errorf("want nil") }
}

func TestApplyDeltas_FillsDeltas(t *testing.T) {
	full := FullAnalysis{Brackets: []BracketAnalysis{{
		Bracket: Bracket{Name: "Divine"},
		Cores:   []HeroStat{{Hero: stratz.Hero{DisplayName: "AM"}, WinRate: 55, PickRate: 7}},
	}}}
	prior := &snapshotSummary{
		Date: time.Now().AddDate(0, 0, -7),
		Stats: map[string]map[string]struct{WR, PR float64}{
			"Divine": {"AM": {WR: 53, PR: 6}},
		},
	}
	ApplyDeltas(&full, prior)
	got := full.Brackets[0].Cores[0]
	if got.WRDelta == nil || *got.WRDelta != 2 { t.Errorf("wrDelta=%v want 2", got.WRDelta) }
	if got.PRDelta == nil || *got.PRDelta != 1 { t.Errorf("prDelta=%v want 1", got.PRDelta) }
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/analysis/... -v
```
Expected: PASS.

- [ ] **Step 4: Create history dir and gitkeep**

```bash
mkdir -p docs/history
touch docs/history/.gitkeep
```

- [ ] **Step 5: Modify `update.sh` to snapshot-and-patch**

Replace contents with:

```bash
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Hand-bumped patch string. Update on patch day.
PATCH="${DOTA_PATCH:-7.40b}"

echo "Building dota-meta..."
go build -o dota-meta ./cmd/dota-meta

if [[ -f docs/data.json ]]; then
  DATE=$(date +%Y-%m-%d)
  cp docs/data.json "docs/history/data-${DATE}.json"
  echo "Archived previous snapshot → docs/history/data-${DATE}.json"
fi

echo "Fetching latest hero stats..."
./dota-meta --html --patch "$PATCH"

echo "Generating Reddit post..."
./dota-meta --output reddit.md --patch "$PATCH"

echo ""
echo "=== Done ==="
echo "Site data:   docs/data.json (updated, patch=$PATCH)"
echo "Reddit post: reddit.md (ready to copy)"
echo ""
echo "To publish:"
echo "  git add docs/data.json docs/history reddit.md && git commit -m 'data: update hero stats' && git push"
```

- [ ] **Step 6: Commit**

```bash
git add internal/analysis/history.go internal/analysis/history_test.go internal/analysis/analysis.go docs/history/.gitkeep update.sh
git commit -m "feat(history): snapshot archive + prior-snapshot delta computation"
```

---

## Task 11 — Momentum tag: weekly-WR slope

STRATZ `winWeek` already gives us 20 weeks per bracket per hero — no waiting. Momentum uses the last 4 in-patch weekly WR values from `HeroStat.WRHistory`. Noise floor: `|slope × 4| < 1.0pp`.

**Files:**
- Create: `internal/analysis/momentum.go`.
- Create: `internal/analysis/momentum_test.go`.
- Modify: `internal/analysis/analysis.go` — apply per-hero momentum inside `analyzeBracket`; pull `SnapshotsInPatch` from the window length used.

**Simplification vs spec:** spec talked about `SnapshotsInPatch` gate assuming weekly update.sh runs. Since STRATZ already returns weekly buckets, we can compute slope directly from `WRHistory` with no snapshot accumulation. `SnapshotsInPatch` still tracked = number of WRHistory points we used.

- [ ] **Step 1: Test**

```go
package analysis

import "testing"

func TestLinearSlope(t *testing.T) {
	// y = 2x + 1, x = 0..3 → slope 2
	got := linearSlope([]float64{1, 3, 5, 7})
	if got != 2 { t.Errorf("slope=%v want 2", got) }
}

func TestTagMomentum(t *testing.T) {
	cases := []struct {
		wrSeries, prSeries []float64
		want               MomentumTag
	}{
		{[]float64{50,51,52,53}, []float64{5,6,7,8}, MomentumRising},  // both up, >1pp
		{[]float64{53,52,51,50}, []float64{5,6,7,8}, MomentumFalling}, // WR↓ PR↑
		{[]float64{50,51,52,53}, []float64{5,5,4,4}, MomentumHidden},  // WR↑ PR flat/down
		{[]float64{53,52,51,50}, []float64{8,7,6,5}, MomentumDying},   // both down
		{[]float64{50,50,50,50}, []float64{5,5,5,5}, MomentumNone},    // flat
		{[]float64{50,50,50},    []float64{5,5,5},   MomentumNone},    // <4 points
	}
	for i, c := range cases {
		got := TagMomentum(c.wrSeries, c.prSeries)
		if got != c.want {
			t.Errorf("case %d: got %q want %q", i, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Implement `momentum.go`**

```go
package analysis

import "math"

type MomentumTag string

const (
	MomentumNone    MomentumTag = ""
	MomentumRising  MomentumTag = "rising"
	MomentumFalling MomentumTag = "falling-off"
	MomentumHidden  MomentumTag = "hidden-gem"
	MomentumDying   MomentumTag = "dying"
)

const momentumWindow = 4
const momentumNoiseFloor = 1.0 // pp across the 4-week projection

// TagMomentum classifies the last `momentumWindow` points of both series.
// Returns MomentumNone if either series is too short or change is below noise.
func TagMomentum(wr, pr []float64) MomentumTag {
	if len(wr) < momentumWindow || len(pr) < momentumWindow { return MomentumNone }
	wrProj := linearSlope(wr[len(wr)-momentumWindow:]) * float64(momentumWindow)
	prProj := linearSlope(pr[len(pr)-momentumWindow:]) * float64(momentumWindow)
	if math.Abs(wrProj) < momentumNoiseFloor && math.Abs(prProj) < momentumNoiseFloor {
		return MomentumNone
	}
	wrUp := wrProj >= momentumNoiseFloor
	wrDown := wrProj <= -momentumNoiseFloor
	prUp := prProj >= momentumNoiseFloor
	prDown := prProj <= -momentumNoiseFloor
	switch {
	case wrUp && prUp:              return MomentumRising
	case wrDown && prUp:            return MomentumFalling
	case wrUp && (prDown || (!prUp && !prDown)): return MomentumHidden
	case wrDown && prDown:          return MomentumDying
	default:                        return MomentumNone
	}
}

// linearSlope returns the slope of y over x=[0..n-1] via least squares.
func linearSlope(y []float64) float64 {
	n := float64(len(y))
	if n < 2 { return 0 }
	var sx, sy, sxy, sxx float64
	for i, v := range y {
		x := float64(i)
		sx += x
		sy += v
		sxy += x * v
		sxx += x * x
	}
	denom := n*sxx - sx*sx
	if denom == 0 { return 0 }
	return (n*sxy - sx*sy) / denom
}
```

Add `Momentum MomentumTag` field on `HeroStat`.

- [ ] **Step 3: Wire momentum into `analyzeBracket`**

We only have WR weekly from STRATZ, not PR weekly (STRATZ `winWeek` has `matchCount` per hero — but we'd need total-matches per week to compute PR). **Simplification:** for v1, call `TagMomentum(wr, wr)` replacing PR series with WR series? No — that would classify everything as Rising/Dying only. Better: derive a per-week PR series from `AggregatedHero.weekly` matchCount divided by a per-week bracket-total matchCount — but `AggregateBrackets` currently discards per-week bracket-totals.

**Fix:** extend `AggregateBrackets` output to also include `WeeklyPR []float64`, computed as hero.weekly.matches / bracketTotal.weekly.matches × 100 per week.

- [ ] **Step 4: Extend `stratz.AggregateBrackets` to produce `WeeklyPR`**

In `internal/api/stratz/bridge.go`:

```go
type AggregatedHero struct {
	HeroID    int
	Picks     int
	Wins      int
	WeeklyWR  []float64
	WeeklyPR  []float64 // % of bracket matches in that week, oldest first
}
```

Add a parallel `weekTotals` map keyed by `week` summing `matchCount` across all rows in `responses`; use `weekTotals[w]/picksPerMatch` as that week's match count (reusing the /10 heuristic isn't right — `matchCount` already is per (hero, week), and summing all heroes in a week ≈ 10 × matches). So weekly matches ≈ `sum(matchCount) / 10`. Implement that.

Update `AggregateBrackets`:

```go
const picksPerMatch = 10
weekTotalPicks := map[int64]int{}
for _, br := range responses {
	for _, w := range br.Weeks { weekTotalPicks[w.Week] += w.MatchCount }
}
// …inside the series build loop:
series := make([]float64, 0, len(weeks))
prSeries := make([]float64, 0, len(weeks))
for _, w := range weeks {
	x := a.weekly[w]
	if x.picks == 0 { series = append(series, 0); prSeries = append(prSeries, 0); continue }
	series = append(series, float64(x.wins)/float64(x.picks)*100)
	totalMatches := weekTotalPicks[w] / picksPerMatch
	if totalMatches == 0 { prSeries = append(prSeries, 0); continue }
	prSeries = append(prSeries, float64(x.picks)/float64(totalMatches)*100)
}
```

Export `AggregatedHero.WeeklyPR`.

Update existing stratz tests to expect the new field (or at least not break). Add one test asserting `WeeklyPR` is populated.

- [ ] **Step 5: Wire into analysis**

In `analyzeBracket` (after loops):

```go
applyMomentum := func(stats []HeroStat, agg map[int]stratz.AggregatedHero) {
	for i := range stats {
		a := agg[stats[i].Hero.ID]
		stats[i].WRHistory = a.WeeklyWR
		stats[i].PRHistory = a.WeeklyPR
		stats[i].Momentum = TagMomentum(a.WeeklyWR, a.WeeklyPR)
	}
}
applyMomentum(ba.Cores, agg)
applyMomentum(ba.Supports, agg)
```

Add `PRHistory []float64` to `HeroStat`.

- [ ] **Step 6: Run tests**

```bash
go test ./... -v
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/analysis internal/api/stratz
git commit -m "feat(analysis): momentum tag from weekly WR/PR slopes"
```

---

## Task 12 — JSON output format rewrite

**Files:**
- Modify: `internal/format/json.go` — emit new shape.
- Modify: `internal/format/format_test.go` — align fixture.

New shape (additive — drop Best/Sleepers/Traps; replace with Cores/Supports):

```jsonc
{
  "generated": "2026-04-23T01:02:03Z",
  "patch": "7.40b",
  "snapshot_date": "2026-04-23",
  "prior_snapshot": "2026-04-16",
  "snapshots_in_patch": 3,
  "heroes": [ /* unchanged-ish, but image URL now uses stratz shortName */ ],
  "analysis": {
    "brackets": {
      "divine": {
        "name": "Divine",
        "cores":    [ { "name":"Visage", "tier":"pocket-pick", "climb":"scales-up",
                        "win_rate":54.2, "pick_rate":2.1, "picks":8200,
                        "wr_delta":0.8, "pr_delta":0.2,
                        "momentum":"hidden-gem",
                        "wr_history":[53.1, 53.5, 53.9, 54.2] } ],
        "supports": [ ]
      }
      /* herald_guardian, crusader_archon, legend_ancient, immortal likewise */
    }
  }
}
```

- [ ] **Step 1: Define new types in `json.go`**

```go
type jsonOutput struct {
	Generated        string       `json:"generated"`
	Patch            string       `json:"patch,omitempty"`
	SnapshotDate     string       `json:"snapshot_date,omitempty"`
	PriorSnapshot    string       `json:"prior_snapshot,omitempty"`
	SnapshotsInPatch int          `json:"snapshots_in_patch"`
	Heroes           []jsonHero   `json:"heroes"`
	Analysis         jsonAnalysis `json:"analysis"`
}

type jsonHero struct {
	Name        string                     `json:"name"`
	ShortName   string                     `json:"short_name"`
	PrimaryAttr string                     `json:"primary_attr"`
	AttackType  string                     `json:"attack_type"`
	Roles       []string                   `json:"roles"`
	Brackets    map[string]jsonBracketStat `json:"brackets"`
}

type jsonBracketStat struct {
	Picks    int     `json:"picks"`
	Wins     int     `json:"wins"`
	WinRate  float64 `json:"win_rate"`
	PickRate float64 `json:"pick_rate"`
}

type jsonAnalysis struct {
	Brackets map[string]jsonBracketAnalysis `json:"brackets"`
}

type jsonBracketAnalysis struct {
	Name     string         `json:"name"`
	Cores    []jsonHeroStat `json:"cores"`
	Supports []jsonHeroStat `json:"supports"`
}

type jsonHeroStat struct {
	Name      string    `json:"name"`
	ShortName string    `json:"short_name"`
	Tier      string    `json:"tier"`
	Climb     string    `json:"climb,omitempty"`
	WinRate   float64   `json:"win_rate"`
	PickRate  float64   `json:"pick_rate"`
	Picks     int       `json:"picks"`
	WRDelta   *float64  `json:"wr_delta,omitempty"`
	PRDelta   *float64  `json:"pr_delta,omitempty"`
	Momentum  string    `json:"momentum,omitempty"`
	WRHistory []float64 `json:"wr_history,omitempty"`
}

var bracketKeyMap = map[string]string{
	"Herald-Guardian": "herald_guardian",
	"Crusader-Archon": "crusader_archon",
	"Legend-Ancient":  "legend_ancient",
	"Divine":          "divine",
	"Immortal":        "immortal",
}
```

- [ ] **Step 2: Rewrite `FormatJSON` signature**

```go
func FormatJSON(heroes []stratz.Hero, result analysis.FullAnalysis) ([]byte, error) {
	output := jsonOutput{
		Generated:        time.Now().UTC().Format(time.RFC3339),
		Patch:            result.Patch,
		SnapshotDate:     result.SnapshotDate.Format("2006-01-02"),
		SnapshotsInPatch: result.SnapshotsInPatch,
		Heroes:           buildJSONHeroes(heroes, result),
		Analysis:         buildJSONAnalysis(result),
	}
	if result.PriorSnapshot != nil {
		output.PriorSnapshot = result.PriorSnapshot.Format("2006-01-02")
	}
	return json.MarshalIndent(output, "", "  ")
}

func buildJSONHeroes(heroes []stratz.Hero, result analysis.FullAnalysis) []jsonHero {
	// Build (heroID → bracketKey → stats) from analysis brackets.
	statLookup := map[int]map[string]jsonBracketStat{}
	for _, ba := range result.Brackets {
		key := bracketKeyMap[ba.Bracket.Name]
		add := func(stats []analysis.HeroStat) {
			for _, s := range stats {
				m, ok := statLookup[s.Hero.ID]
				if !ok { m = map[string]jsonBracketStat{}; statLookup[s.Hero.ID] = m }
				m[key] = jsonBracketStat{
					Picks: s.Picks, Wins: s.Wins, WinRate: s.WinRate, PickRate: s.PickRate,
				}
			}
		}
		add(ba.Cores); add(ba.Supports)
	}
	out := make([]jsonHero, 0, len(heroes))
	for _, h := range heroes {
		out = append(out, jsonHero{
			Name:        h.DisplayName,
			ShortName:   h.ShortName,
			PrimaryAttr: h.PrimaryAttribute,
			AttackType:  h.AttackType,
			Roles:       h.Roles,
			Brackets:    statLookup[h.ID],
		})
	}
	return out
}

func buildJSONAnalysis(result analysis.FullAnalysis) jsonAnalysis {
	ja := jsonAnalysis{Brackets: make(map[string]jsonBracketAnalysis, len(result.Brackets))}
	for _, ba := range result.Brackets {
		key := bracketKeyMap[ba.Bracket.Name]
		ja.Brackets[key] = jsonBracketAnalysis{
			Name:     ba.Bracket.Name,
			Cores:    heroStatsToJSON(ba.Cores),
			Supports: heroStatsToJSON(ba.Supports),
		}
	}
	return ja
}

func heroStatsToJSON(stats []analysis.HeroStat) []jsonHeroStat {
	out := make([]jsonHeroStat, 0, len(stats))
	for _, s := range stats {
		js := jsonHeroStat{
			Name:      s.Hero.DisplayName,
			ShortName: s.Hero.ShortName,
			Tier:      s.Tier.String(),
			WinRate:   s.WinRate,
			PickRate:  s.PickRate,
			Picks:     s.Picks,
			WRDelta:   s.WRDelta,
			PRDelta:   s.PRDelta,
			WRHistory: s.WRHistory,
		}
		if s.ClimbTag != analysis.ClimbUnknown { js.Climb = string(s.ClimbTag) }
		if s.Momentum != analysis.MomentumNone { js.Momentum = string(s.Momentum) }
		out = append(out, js)
	}
	return out
}
```

- [ ] **Step 3: Update `format_test.go`** — rewrite to exercise the new shape. One test ensures Tier/Climb/Momentum strings survive round-trip and PriorSnapshot is only emitted when set.

```go
package format

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

func TestFormatJSON_EmitsTierClimbMomentum(t *testing.T) {
	heroes := []stratz.Hero{{ID: 1, DisplayName: "Visage", ShortName: "visage", Roles: []string{"Carry"}}}
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	full := analysis.FullAnalysis{
		Patch: "7.40b",
		SnapshotDate: now,
		Brackets: []analysis.BracketAnalysis{{
			Bracket: analysis.Bracket{Name: "Divine"},
			Cores: []analysis.HeroStat{{
				Hero: heroes[0], Picks: 1200, Wins: 660, WinRate: 55, PickRate: 2.3,
				Tier: analysis.TierPocketPick, ClimbTag: analysis.ClimbUp,
				Momentum: analysis.MomentumHidden, WRHistory: []float64{53, 54, 54.5, 55},
			}},
		}},
	}
	out, err := FormatJSON(heroes, full)
	if err != nil { t.Fatal(err) }
	var got map[string]any
	json.Unmarshal(out, &got)
	a := got["analysis"].(map[string]any)["brackets"].(map[string]any)["divine"].(map[string]any)
	cores := a["cores"].([]any)
	core0 := cores[0].(map[string]any)
	if core0["tier"] != "pocket-pick" { t.Errorf("tier=%v", core0["tier"]) }
	if core0["climb"] != "scales-up" { t.Errorf("climb=%v", core0["climb"]) }
	if core0["momentum"] != "hidden-gem" { t.Errorf("momentum=%v", core0["momentum"]) }
	if !strings.Contains(string(out), `"wr_history"`) { t.Errorf("no wr_history in output") }
}
```

- [ ] **Step 4: Run**

```bash
go test ./internal/format/... -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/format/
git commit -m "feat(format): emit tier/climb/momentum/deltas/history in JSON"
```

---

## Task 13 — Reddit post rewrite

**Files:**
- Modify: `internal/format/reddit.go`.
- Modify: `internal/format/format_test.go` (reddit test).

Target sections per spec:
- Intro with match count + patch.
- Per bracket: "Ban list" (top 3 Meta Tyrants by WR × PR), "Pocket picks" (top 3 Pocket Picks by WR), "Stop picking" (top 3 Traps by PR).
- "Rising this week" section across all brackets — only included when ≥1 hero has `MomentumRising`.
- Keep it short — one line per hero in text form, no big tables.

- [ ] **Step 1: Rewrite `FormatReddit`**

Key helpers:

```go
func topN[T any](xs []T, n int, less func(a, b T) bool) []T {
	sort.Slice(xs, func(i, j int) bool { return less(xs[i], xs[j]) })
	if len(xs) > n { xs = xs[:n] }
	return xs
}
func filterTier(stats []analysis.HeroStat, tier analysis.Tier) []analysis.HeroStat {
	out := stats[:0:0]
	for _, s := range stats { if s.Tier == tier { out = append(out, s) } }
	return out
}
```

```go
func FormatReddit(result analysis.FullAnalysis, date string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## [Weekly] Heroes that are secretly broken at your rank -- Week of %s\n\n", date)
	fmt.Fprintf(&b, "Patch %s · %s matches analyzed.\n\n", result.Patch, formatNumber(result.TotalMatches))
	b.WriteString("Tiers are computed per-bracket from median win rate and pick rate splits — so the same hero can be a Meta Tyrant in Divine and a Trap in Herald. ≥1000 picks per bracket to qualify.\n\n---\n\n")

	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "### %s\n\n", ba.Bracket.Name)
		writeSection(&b, "Ban list (Meta Tyrants)", filterTier(append(append([]analysis.HeroStat{}, ba.Cores...), ba.Supports...), analysis.TierMetaTyrant),
			func(a, b analysis.HeroStat) bool { return a.WinRate*a.PickRate > b.WinRate*b.PickRate })
		writeSection(&b, "Pocket picks (sleepers)", filterTier(append(append([]analysis.HeroStat{}, ba.Cores...), ba.Supports...), analysis.TierPocketPick),
			func(a, b analysis.HeroStat) bool { return a.WinRate > b.WinRate })
		writeSection(&b, "Stop picking (Traps)", filterTier(append(append([]analysis.HeroStat{}, ba.Cores...), ba.Supports...), analysis.TierTrap),
			func(a, b analysis.HeroStat) bool { return a.PickRate > b.PickRate })
	}

	rising := collectRising(result)
	if len(rising) > 0 {
		b.WriteString("### Rising this week\n\n")
		for _, s := range rising {
			fmt.Fprintf(&b, "- **%s** — %.1f%% WR, %.1f%% PR (slope up in both)\n", s.Hero.DisplayName, s.WinRate, s.PickRate)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func writeSection(b *strings.Builder, title string, heroes []analysis.HeroStat, cmp func(a, b analysis.HeroStat) bool) {
	fmt.Fprintf(b, "**%s:** ", title)
	if len(heroes) == 0 { b.WriteString("none this week.\n\n"); return }
	top := topN(heroes, 3, cmp)
	parts := make([]string, len(top))
	for i, s := range top {
		parts[i] = fmt.Sprintf("%s (%.1f%% WR, %.1f%% PR)", s.Hero.DisplayName, s.WinRate, s.PickRate)
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteString("\n\n")
}

func collectRising(result analysis.FullAnalysis) []analysis.HeroStat {
	seen := map[string]bool{}
	var out []analysis.HeroStat
	for _, ba := range result.Brackets {
		for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
			for _, s := range list {
				if s.Momentum != analysis.MomentumRising { continue }
				k := ba.Bracket.Name + "|" + s.Hero.DisplayName
				if seen[k] { continue }
				seen[k] = true
				out = append(out, s)
			}
		}
	}
	return out
}
```

- [ ] **Step 2: Update reddit test**

```go
func TestFormatReddit_HasBanListAndPocketSections(t *testing.T) {
	full := analysis.FullAnalysis{
		Patch: "7.40b", TotalMatches: 100000,
		Brackets: []analysis.BracketAnalysis{{
			Bracket: analysis.Bracket{Name: "Divine"},
			Cores: []analysis.HeroStat{
				{Hero: stratz.Hero{DisplayName: "Visage"}, WinRate: 55, PickRate: 5, Tier: analysis.TierMetaTyrant},
				{Hero: stratz.Hero{DisplayName: "Sniper"}, WinRate: 56, PickRate: 1, Tier: analysis.TierPocketPick},
				{Hero: stratz.Hero{DisplayName: "PA"}, WinRate: 45, PickRate: 10, Tier: analysis.TierTrap},
			},
		}},
	}
	post := FormatReddit(full, "April 23, 2026")
	if !strings.Contains(post, "Ban list") { t.Error("missing Ban list") }
	if !strings.Contains(post, "Pocket picks") { t.Error("missing Pocket picks") }
	if !strings.Contains(post, "Stop picking") { t.Error("missing Stop picking") }
	if !strings.Contains(post, "7.40b") { t.Error("missing patch") }
}
```

- [ ] **Step 3: Run**

```bash
go test ./internal/format/... -v
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/format/
git commit -m "feat(reddit): rewrite post around tier sections and momentum"
```

---

## Task 14 — Wire STRATZ + history into `cmd/dota-meta/main.go`

**Files:**
- Modify: `cmd/dota-meta/main.go`.

- [ ] **Step 1: Add `--patch` flag, STRATZ env var, load prior snapshot**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
	"github.com/narrowcastdev/dota-meta/internal/format"
)

func main() {
	outputFile := flag.String("output", "", "write Reddit post to file instead of stdout")
	jsonMode := flag.Bool("json", false, "output raw analysis as JSON")
	htmlMode := flag.Bool("html", false, "generate docs/data.json for static site")
	minPicks := flag.Int("min-picks", 1000, "minimum picks to qualify a hero")
	patch := flag.String("patch", "", "current Dota patch version (e.g. 7.40b)")
	flag.Parse()

	token := os.Getenv("STRATZ_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "STRATZ_TOKEN env var not set")
		os.Exit(1)
	}
	client := stratz.NewClient(token)
	heroes, brackets, err := client.FetchAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetching stratz: %v\n", err)
		os.Exit(1)
	}

	result := analysis.Analyze(heroes, brackets, *minPicks)
	result.Patch = *patch
	result.SnapshotDate = time.Now().UTC()

	if prior, err := analysis.LoadLatestPriorSnapshot("docs/history", result.SnapshotDate); err == nil {
		analysis.ApplyDeltas(&result, prior)
	} else {
		fmt.Fprintf(os.Stderr, "warn: history load failed: %v\n", err)
	}

	date := result.SnapshotDate.Format("January 2, 2006")

	if *jsonMode {
		data, jsonErr := format.FormatJSON(heroes, result)
		if jsonErr != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", jsonErr)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	if *htmlMode {
		data, jsonErr := format.FormatJSON(heroes, result)
		if jsonErr != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", jsonErr)
			os.Exit(1)
		}
		if writeErr := os.WriteFile("docs/data.json", data, 0644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing docs/data.json: %v\n", writeErr)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Wrote docs/data.json")
	}

	post := format.FormatReddit(result, date)
	if *outputFile != "" {
		if writeErr := os.WriteFile(*outputFile, []byte(post), 0644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *outputFile, writeErr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %s\n", *outputFile)
		return
	}
	fmt.Print(post)
}
```

- [ ] **Step 2: Build**

```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 3: Dry-run with fake token (should fail on network)**

```bash
STRATZ_TOKEN=dummy ./dota-meta --json >/dev/null
```
Expected: non-zero exit (STRATZ rejects dummy token). That's fine — confirms wiring.

- [ ] **Step 4: Commit**

```bash
git add cmd/dota-meta/main.go
git commit -m "feat(cmd): drive analysis from stratz + history snapshots"
```

---

## Task 15 — Frontend: grids, chips, deltas, legend

**Files:**
- Modify: `docs/index.html`, `docs/app.js`, `docs/style.css`.

- [ ] **Step 1: Rewrite `app.js` around new shape**

Read the first 30 lines of the existing `docs/app.js` to keep the data-load pattern, then replace the render code.

```bash
head -n 60 docs/app.js
```

The new file should:
- Fetch `data.json`.
- Render a bracket selector (same as existing) with STRATZ's 5 brackets.
- Render a role selector: Cores / Supports / All.
- For each role section, render a 2×2 grid of tiers (Meta Tyrant, Pocket Pick, Trap, Dead). Dead is collapsed by default.
- Each tier cell lists hero chips sorted by WinRate desc.
- Hero chip: `<name> [<climb icon>] <WR%> [<±wrDelta>] <PR%> [<momentum icon>]`. Hover renders inline SVG sparkline of last 8 `wr_history` points.
- Header shows patch + snapshot date + (if present) "vs <prior_snapshot>".

Pseudocode structure:

```js
const MOMENTUM_ICON = { rising: "🔥", "falling-off": "⚠️", "hidden-gem": "💤", dying: "📉" };
const CLIMB_ICON = { "scales-up": "↗", "scales-down": "↘", "universal": "≈" };
const TIER_ORDER = ["meta-tyrant", "pocket-pick", "trap", "dead"];
const TIER_TITLE = {
  "meta-tyrant": "Meta Tyrant — ban or first-pick",
  "pocket-pick": "Pocket Pick — last-pick counter",
  "trap":        "Trap — popular but losing",
  "dead":        "Dead — ignore",
};
const BRACKET_KEYS = ["herald_guardian","crusader_archon","legend_ancient","divine","immortal"];

async function main() {
  const data = await fetch("data.json").then(r => r.json());
  renderHeader(data);
  wireBracketAndRoleSelectors(data);
  renderLegend();
  renderBracket(data, currentBracket(), currentRole());
}

function renderBracket(data, bracketKey, role) {
  const ba = data.analysis.brackets[bracketKey];
  const container = document.getElementById("tiers");
  container.innerHTML = "";
  const sections = role === "all" ? ["cores","supports"] : [role];
  for (const sec of sections) {
    container.appendChild(renderRoleSection(sec, ba[sec] || []));
  }
}

function renderRoleSection(label, stats) { /* groups stats by tier, builds .tier-cell divs */ }
function renderChip(stat) { /* name + icons + deltas + sparkline on hover */ }
function sparklineSVG(series, w=80, h=24) { /* polyline with min/max scaling */ }
```

Keep the whole file under ~250 lines. No framework.

- [ ] **Step 2: Update `index.html`**

Add container `<div id="tiers"></div>`, role selector `<select id="role"><option value="all">All</option>…</select>`, legend `<div id="legend"></div>`. Header: patch + snapshot.

- [ ] **Step 3: Add styles in `style.css`** — `.tier-grid` (2×2 CSS grid), `.tier-cell`, `.chip`, `.chip:hover .sparkline`, `.tier-cell.dead` collapsed.

- [ ] **Step 4: Manual smoke test**

```bash
cd docs && python3 -m http.server 8080 &
open http://localhost:8080
```

Using a hand-crafted mini `data.json` (or the last real one), verify:
- Bracket selector works for all 5.
- Role switch shows cores, supports, all.
- Climb icons appear next to hero names.
- Delta numbers with ± prefix appear when wr_delta present.
- Dead section starts collapsed; clicks to expand.
- Hover shows sparkline for heroes with `wr_history`.
- Momentum icons appear for heroes tagged.

- [ ] **Step 5: Commit**

```bash
git add docs/index.html docs/app.js docs/style.css
git commit -m "feat(web): tier grids, chips, sparklines, deltas, legend"
```

---

## Task 16 — End-to-end integration check + README note

**Files:**
- Modify: `README.md` — document `STRATZ_TOKEN`, `update.sh`, history dir layout.

- [ ] **Step 1: Run the full pipeline locally against real STRATZ**

```bash
export STRATZ_TOKEN=<real token from 1Password>
./update.sh
```
Expected: `docs/data.json` regenerated, `docs/history/data-<today>.json` archived (if a prior `data.json` existed), `reddit.md` generated.

- [ ] **Step 2: Inspect output**

- `jq '.patch, .snapshot_date, (.analysis.brackets | keys)' docs/data.json` — shows 5 bracket keys.
- `jq '.analysis.brackets.divine.cores[0]' docs/data.json` — contains tier, climb, momentum, wr_history.
- Open the site locally, confirm rendering.

- [ ] **Step 3: Update README**

Add a "Running" section noting:
- Needs `STRATZ_TOKEN` in env.
- Patch bumped via `DOTA_PATCH=7.40c ./update.sh` or edit `update.sh`.
- History lives under `docs/history/data-YYYY-MM-DD.json` — safe to commit.
- OpenDota client archived under `internal/api/opendota/`.

- [ ] **Step 4: Final formatting + full test run**

```bash
gofmt -s -w .
go vet ./...
go test ./...
```
Expected: all PASS, no formatting diff.

- [ ] **Step 5: Commit**

```bash
git add README.md docs/data.json docs/history/ reddit.md
git commit -m "docs(dota-meta): document stratz migration + history workflow"
```

---

## Post-plan checklist (spec coverage)

- Tier quadrants per bracket, per role family → Task 8.
- Medians not fixed thresholds → Task 8 (`median()` helper).
- Min-picks floor 1000 → Task 7 (`analyzeBracket`).
- Climb tag (scales up/down/universal) at ±2pp → Task 9.
- STRATZ primary, OpenDota archived → Tasks 1-5.
- Immortal as its own bracket → Task 6 (5-bracket list).
- Snapshot history in `docs/history/data-YYYY-MM-DD.json` with patch field → Task 10.
- WR/PR deltas from prior snapshot → Task 10.
- Momentum tag (rising / falling-off / hidden-gem / dying) with noise floor → Task 11.
- Sparklines on hover → Task 15 (inline SVG, no lib).
- Reddit format: Ban list / Pocket picks / Stop picking / Rising → Task 13.
- Legend on UI → Task 15.
- Dead tier collapsed → Task 15.

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-23-tiers-trends.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
