// Command fetchphotos rebuilds the local photo cache from the fixture
// file. Shelter photos are never committed, a fresh clone runs this once.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nazboyko/good-dog/internal/curation"
)

func main() {
	path := "fixtures/dogs.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	dogs, err := curation.LoadDogs(path)
	if err != nil {
		fatal("%v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var problems []string
	hashes := map[string]string{}
	for _, d := range dogs {
		if d.PhotoLocal == "" {
			continue
		}
		if _, err := os.Stat(d.PhotoLocal); err != nil {
			if d.PhotoURL == "" {
				problems = append(problems, fmt.Sprintf("no photo_url for %s (%s)", d.Name, d.ID))
				continue
			}
			if err := download(client, d.PhotoURL, d.PhotoLocal); err != nil {
				problems = append(problems, fmt.Sprintf("fetch %s (%s): %v", d.Name, d.ID, err))
				continue
			}
			fmt.Printf("fetched %s (%s)\n", d.Name, d.PhotoLocal)
		} else {
			fmt.Printf("have %s (%s)\n", d.Name, d.PhotoLocal)
		}
		sum, err := curation.HashFile(d.PhotoLocal)
		if err != nil {
			problems = append(problems, fmt.Sprintf("hash %s (%s): %v", d.Name, d.ID, err))
			continue
		}
		hashes[d.ID] = sum
	}

	m, err := curation.LoadManifest(curation.ManifestPath)
	if err != nil {
		fatal("read manifest: %v", err)
	}
	problems = append(problems, curation.Duplicates(hashes)...)
	problems = append(problems, curation.Changed(m, hashes)...)
	// a mismatch is never written back, it would erase itself next run
	if err := curation.SaveManifest(curation.ManifestPath, m, hashes); err != nil {
		fatal("write manifest: %v", err)
	}

	for _, p := range problems {
		fmt.Fprintln(os.Stderr, p)
	}
	if len(problems) > 0 {
		fatal("%d photo problems, fix them before these dogs reach a player", len(problems))
	}
}

// download writes to a temp file first so an interrupted run never leaves
// a half photo that the next run would treat as cached.
func download(client *http.Client, url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %s", resp.Status)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".partial-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// CreateTemp makes 0600, a cache file the whole toolchain reads
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
