// Command verifyfixtures checks each curated dog against the live web:
// the listing still answers and still names the dog, and the cached photo
// still matches the manifest. It only reads. A dog that fails is a hand
// recheck and a curated edit, never an automatic status flip.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nazboyko/good-dog/internal/curation"
)

// one request per dog, spaced out, with a user agent that says who we are
const (
	pause     = 2 * time.Second
	userAgent = "good-dog fixture verifier (weekend project, one request per dog)"
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
	manifest, err := curation.LoadManifest(curation.ManifestPath)
	if err != nil {
		fatal("read manifest: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var recheck []string
	for i, d := range dogs {
		if i > 0 {
			time.Sleep(pause)
		}
		verdict, ok := check(client, d, manifest)
		mark := "ok  "
		if !ok {
			mark = "FAIL"
			recheck = append(recheck, fmt.Sprintf("%s (%s) %s", d.Name, d.ID, d.ListingURL))
		}
		fmt.Printf("%s %-11s %-14s %s\n", mark, d.Name, d.ID, verdict)
	}

	fmt.Printf("\n%d dogs, %d to recheck\n", len(dogs), len(recheck))
	if len(recheck) == 0 {
		return
	}
	fmt.Println("\nopen these by hand:")
	for _, line := range recheck {
		fmt.Println("  " + line)
	}
	fmt.Println("\nA dog that fails has not necessarily left. Read the listing, then make a")
	fmt.Println("curated edit to fixtures/dogs.json with a devlog line. This tool never")
	fmt.Println("changes status by itself.")
	os.Exit(1)
}

func check(client *http.Client, d curation.Dog, m curation.Manifest) (string, bool) {
	var notes []string
	ok := true

	switch status, body, err := get(client, d.ListingURL); {
	case err != nil:
		notes = append(notes, "listing unreachable: "+err.Error())
		ok = false
	case status != http.StatusOK:
		notes = append(notes, fmt.Sprintf("listing http %d", status))
		ok = false
	case !namedIn(d.Name, body):
		notes = append(notes, fmt.Sprintf("listing 200 but does not name %q", d.Name))
		ok = false
	default:
		notes = append(notes, "listing 200, name found")
	}

	switch sum, err := curation.HashFile(d.PhotoLocal); {
	case err != nil:
		notes = append(notes, "photo missing from cache, run make fetch-photos")
		ok = false
	case m.Photos[d.ID] == "":
		notes = append(notes, "photo not in manifest yet")
	case m.Photos[d.ID] != sum:
		notes = append(notes, fmt.Sprintf("photo changed, manifest %s file %s",
			curation.Short(m.Photos[d.ID]), curation.Short(sum)))
		ok = false
	default:
		notes = append(notes, "photo matches")
	}

	return strings.Join(notes, ", "), ok
}

func get(client *http.Client, url string) (int, string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return resp.StatusCode, string(body), err
}

var (
	tags   = regexp.MustCompile(`<[^>]*>`)
	spaces = regexp.MustCompile(`(?:&nbsp;|[\s\x{00A0}])+`)
)

// namedIn matches what a reader would see: tags become spaces, so a name
// split across markup still counts, and case and spacing stop mattering.
// Loose on purpose, a false alarm costs one hand check, a false all clear
// costs a player being told an adopted dog is still waiting.
func namedIn(name, body string) bool {
	flatten := func(s string) string {
		s = tags.ReplaceAllString(strings.ToLower(s), " ")
		return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
	}
	return strings.Contains(flatten(body), flatten(name))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
