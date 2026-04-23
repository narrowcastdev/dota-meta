package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
	"github.com/narrowcastdev/dota-meta/internal/format"
)

// loadDotenv reads KEY=VALUE lines from path and sets any keys not already
// present in the process environment. Silently no-ops if path is missing.
// Supports `#` comment lines, blank lines, and optional surrounding quotes on
// the value. Not a general shell parser.
func loadDotenv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func main() {
	outputFile := flag.String("output", "", "write Reddit post to file instead of stdout")
	jsonMode := flag.Bool("json", false, "output raw analysis as JSON")
	htmlMode := flag.Bool("html", false, "generate docs/data.json for static site")
	minPicks := flag.Int("min-picks", 1000, "minimum picks to qualify a hero")
	patch := flag.String("patch", "", "current Dota patch version (e.g. 7.40b)")
	flag.Parse()

	if err := loadDotenv(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "warn: loading .env: %v\n", err)
	}

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
