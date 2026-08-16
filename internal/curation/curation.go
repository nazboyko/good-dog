// Package curation holds the pieces the fixture tooling shares: reading
// the curated file and keeping the photo cache honest. Tooling only, the
// game never imports this.
package curation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Dog is the slice of a fixture entry the tooling needs. The game reads
// the same file through animal.FixtureProvider and its own types.
type Dog struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PhotoURL   string `json:"photo_url"`
	PhotoLocal string `json:"photo_local"`
	ListingURL string `json:"listing_url"`
	// what the fixture claims right now, so the verifier can hold it up
	// against what the listing itself says
	Status    string `json:"status"`
	Synthetic bool   `json:"synthetic"`
}

// LoadDogs returns the real dogs only. Example entries point at urls that
// were never meant to resolve.
func LoadDogs(path string) ([]Dog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixtures: %w", err)
	}
	var file struct {
		Dogs []Dog `json:"dogs"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("parse fixtures: %w", err)
	}
	var dogs []Dog
	for _, d := range file.Dogs {
		if !d.Synthetic {
			dogs = append(dogs, d)
		}
	}
	return dogs, nil
}

// Manifest remembers what each dog's photo hashed to, so a later run can
// tell a cache rebuild from a photo that quietly changed underneath us.
type Manifest struct {
	Version int               `json:"version"`
	Photos  map[string]string `json:"photos"`
}

const ManifestPath = "cache/photos/manifest.json"

func LoadManifest(path string) (Manifest, error) {
	m := Manifest{Version: 1, Photos: map[string]string{}}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	if m.Photos == nil {
		m.Photos = map[string]string{}
	}
	return m, nil
}

// SaveManifest only fills in dogs the manifest has never seen. An entry
// that disagrees with the file stays, so the mismatch keeps reporting.
func SaveManifest(path string, m Manifest, hashes map[string]string) error {
	m.Version = 1
	for id, sum := range hashes {
		if _, known := m.Photos[id]; !known {
			m.Photos[id] = sum
		}
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// Duplicates catches the failure nothing else sees: a real url that
// serves another dog's face. Both ids are named so the fix is obvious.
func Duplicates(hashes map[string]string) []string {
	byHash := map[string][]string{}
	for id, sum := range hashes {
		byHash[sum] = append(byHash[sum], id)
	}
	var problems []string
	for sum, ids := range byHash {
		if len(ids) < 2 {
			continue
		}
		sort.Strings(ids)
		problems = append(problems, fmt.Sprintf(
			"dogs %v share one photo (sha256 %s), at most one of them can be right", ids, Short(sum)))
	}
	sort.Strings(problems)
	return problems
}

// Changed reports cached files that no longer match the manifest.
func Changed(m Manifest, hashes map[string]string) []string {
	var problems []string
	for id, sum := range hashes {
		known, ok := m.Photos[id]
		if ok && known != sum {
			problems = append(problems, fmt.Sprintf(
				"photo for %s changed on disk, manifest has %s, file is %s", id, Short(known), Short(sum)))
		}
	}
	sort.Strings(problems)
	return problems
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Short trims a hash to the length a person can compare by eye.
func Short(sum string) string {
	if len(sum) <= 12 {
		return sum
	}
	return sum[:12]
}
