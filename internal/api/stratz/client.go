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
