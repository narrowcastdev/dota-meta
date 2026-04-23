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
