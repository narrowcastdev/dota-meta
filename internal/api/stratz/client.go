package stratz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	endpoint = "https://api.stratz.com/graphql"
	// userAgent is required by STRATZ's Cloudflare layer; requests without
	// `STRATZ_API` in the UA get challenged with a 403.
	userAgent = "STRATZ_API"
)

// Client talks to STRATZ GraphQL.
type Client struct {
	Token    string
	HTTP     *http.Client
	Endpoint string // defaults to stratz production; overridable in tests
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
	req.Header.Set("User-Agent", userAgent)

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

// roleIDToName maps STRATZ's `HeroRole` enum values to the friendlier strings
// the analysis layer expects (notably "Carry" and "Support", which drive the
// core/support split).
var roleIDToName = map[string]string{
	"CARRY":     "Carry",
	"DISABLER":  "Disabler",
	"DURABLE":   "Durable",
	"ESCAPE":    "Escape",
	"INITIATOR": "Initiator",
	"NUKER":     "Nuker",
	"PUSHER":    "Pusher",
	"SUPPORT":   "Support",
	"JUNGLER":   "Jungler",
}

// FetchHeroes returns the hero catalog. Call once per run.
func (c *Client) FetchHeroes() ([]Hero, error) {
	body, _ := json.Marshal(graphQLRequest{Query: heroesQuery})
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building heroes request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("User-Agent", userAgent)
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

func parseHeroes(r io.Reader) ([]Hero, error) {
	var raw struct {
		Data struct {
			Constants struct {
				Heroes []struct {
					ID          int    `json:"id"`
					ShortName   string `json:"shortName"`
					DisplayName string `json:"displayName"`
					Roles       []struct {
						RoleID string `json:"roleId"`
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
			time.Sleep(200 * time.Millisecond)
		}
		br, err := c.FetchBracket(b)
		if err != nil {
			return nil, nil, fmt.Errorf("bracket %s: %w", b, err)
		}
		brackets = append(brackets, br)
	}
	return heroes, brackets, nil
}
