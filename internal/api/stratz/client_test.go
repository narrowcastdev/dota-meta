package stratz

import (
	"bytes"
	"io"
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

func TestFetchHeroes_ParsesFixture(t *testing.T) {
	fixture, _ := os.ReadFile("testdata/heroes.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()
	c := NewClient("x")
	c.Endpoint = srv.URL
	got, _, err := c.FetchHeroes()
	if err != nil {
		t.Fatalf("FetchHeroes: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].DisplayName != "Anti-Mage" {
		t.Errorf("bad name: %s", got[0].DisplayName)
	}
	if !contains(got[0].Roles, "Carry") {
		t.Errorf("missing Carry: %v", got[0].Roles)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

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
	c := NewClient("x")
	c.Endpoint = srv.URL
	heroes, _, brackets, err := c.FetchAll()
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(heroes) == 0 {
		t.Error("no heroes")
	}
	if len(brackets) != 8 {
		t.Errorf("brackets=%d, want 8", len(brackets))
	}
	if count != 9 {
		t.Errorf("http calls=%d, want 9", count)
	}
}
