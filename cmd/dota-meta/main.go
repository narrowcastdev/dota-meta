package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
	"github.com/narrowcastdev/dota-meta/internal/format"
)

// archivePriorSnapshot copies dataPath into historyDir named by the
// snapshot_date field already inside the file, so archives never collide with
// today's regenerated output. Skips if archive file already exists.
func archivePriorSnapshot(dataPath, historyDir string) error {
	data, err := os.ReadFile(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var head struct {
		SnapshotDate string `json:"snapshot_date"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return fmt.Errorf("parsing %s: %w", dataPath, err)
	}
	if head.SnapshotDate == "" {
		return nil
	}
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}
	dest := filepath.Join(historyDir, "data-"+head.SnapshotDate+".json")
	if _, err := os.Stat(dest); err == nil {
		return nil // already archived
	}
	return os.WriteFile(dest, data, 0644)
}

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
	infographicFile := flag.String("infographic", "", "write per-bracket infographic PNGs to directory")
	useImages := flag.Bool("images", false, "use image placeholders in Reddit post instead of tables")
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
	heroes, detectedPatch, brackets, err := client.FetchAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetching stratz: %v\n", err)
		os.Exit(1)
	}

	result := analysis.Analyze(heroes, brackets, *minPicks)
	if *patch != "" {
		result.Patch = *patch
	} else {
		result.Patch = detectedPatch
	}
	result.SnapshotDate = time.Now().UTC()
	// WR/PR deltas now come from STRATZ week-over-week inside Analyze. Mark the
	// prior snapshot as exactly one week ago for UI copy.
	prior := result.SnapshotDate.AddDate(0, 0, -7)
	result.PriorSnapshot = &prior

	date := result.SnapshotDate.Format("January 2, 2006")

	if *infographicFile != "" {
		if err := os.MkdirAll(*infographicFile, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", *infographicFile, err)
			os.Exit(1)
		}
		icons := fetchHeroIcons(result)
		fmt.Fprintf(os.Stderr, "Fetched %d hero icons\n", len(icons))

		images, igErr := format.FormatBracketImages(result, date, icons)
		if igErr != nil {
			fmt.Fprintf(os.Stderr, "Error generating infographics: %v\n", igErr)
			os.Exit(1)
		}
		for _, bi := range images {
			path := filepath.Join(*infographicFile, bi.Slug+".png")
			f, fErr := os.Create(path)
			if fErr != nil {
				fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, fErr)
				os.Exit(1)
			}
			if fErr = png.Encode(f, bi.Image); fErr != nil {
				f.Close()
				fmt.Fprintf(os.Stderr, "Error encoding %s: %v\n", path, fErr)
				os.Exit(1)
			}
			f.Close()
			fmt.Fprintf(os.Stderr, "Wrote %s\n", path)
		}
	}

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
		if err := archivePriorSnapshot("docs/data.json", "docs/history"); err != nil {
			fmt.Fprintf(os.Stderr, "warn: archiving prior snapshot: %v\n", err)
		}
		if writeErr := os.WriteFile("docs/data.json", data, 0644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing docs/data.json: %v\n", writeErr)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Wrote docs/data.json")
	}

	var post string
	if *useImages {
		post = format.FormatRedditWithImages(result, date)
	} else {
		post = format.FormatReddit(result, date)
	}

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

func fetchHeroIcons(result analysis.FullAnalysis) map[string]image.Image {
	names := make(map[string]bool)
	for _, ba := range result.Brackets {
		for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
			for _, s := range list {
				if s.Hero.ShortName != "" {
					names[s.Hero.ShortName] = true
				}
			}
		}
	}

	icons := make(map[string]image.Image, len(names))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	client := &http.Client{Timeout: 5 * time.Second}

	for name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			url := "https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/" + n + ".png"
			resp, err := client.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return
			}
			img, _, err := image.Decode(resp.Body)
			if err != nil {
				return
			}
			mu.Lock()
			icons[n] = img
			mu.Unlock()
		}(name)
	}
	wg.Wait()
	return icons
}
