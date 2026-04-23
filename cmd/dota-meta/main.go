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
