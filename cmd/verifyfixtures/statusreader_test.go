package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The status reader is the one guard whose entire job is independent
// ground truth: it exists because a fixture said ACTIVE while the page
// said adopted, and the check passed. So it is tested against snippets
// of the real pages, not against strings written to make it pass.
func TestStatusSaysReadsTheRealPages(t *testing.T) {
	cases := []struct {
		file, want string
		found      bool
	}{
		{"ahs-adopted.html", "listing says adopted on August 15, 2026", true},
		{"ahs-listed.html", "listing says still available", true},
		{"rsmn-listed.html", "listing says still available", true},
		{"nothing.html", "", false},
	}
	for _, c := range cases {
		body, err := os.ReadFile(filepath.Join("testdata", c.file))
		if err != nil {
			t.Fatal(err)
		}
		got, found := statusSays(string(body))
		if found != c.found || got != c.want {
			t.Errorf("%s: got %q %v, want %q %v", c.file, got, found, c.want, c.found)
		}
	}
}

// The adopted page still carries a photo whose alt text says "currently
// available for adoption", stamped from July. It sits inside a tag, so
// the reader must strip tags before it reads, or the stale attribute
// outvotes the sentence the shelter actually wrote. This pins that the
// reader flattens first, by feeding it the attribute as a bare tag.
func TestAStaleAltTextCannotOutvoteTheAdoptionNotice(t *testing.T) {
	page := `<img alt="Bella is currently available for adoption! 7/23/2026">` +
		`<p>We think Bella is pretty great, too, but she is no longer available for adoption. ` +
		`Bella was adopted on August 15, 2026!</p>`
	got, found := statusSays(page)
	if !found || got != "listing says adopted on August 15, 2026" {
		t.Fatalf("read %q, the stale alt won", got)
	}
}

// If the shelter ever writes both the adoption notice and the ordinary
// availability line in visible text, adopted has to win: it is the more
// specific claim and the one that changes what the game says. This is
// hand built because no real page does it yet, and it says so.
func TestAdoptedWinsWhenBothAreVisible(t *testing.T) {
	page := `<p>Bella is available for adoption at Animal Humane Society</p>` +
		`<p>Bella was adopted on August 15, 2026!</p>`
	got, _ := statusSays(page)
	if got != "listing says adopted on August 15, 2026" {
		t.Fatalf("with both visible the reader said %q", got)
	}
}

func TestAgreesIsTheNarrowQuestion(t *testing.T) {
	cases := []struct {
		say, status string
		want        bool
	}{
		{"listing says adopted on August 15, 2026", "ADOPTED_CONFIRMED", true},
		{"listing says adopted on August 15, 2026", "ACTIVE", false},
		{"listing says still available", "ACTIVE", true},
		{"listing says still available", "ADOPTED_CONFIRMED", false},
		{"listing says no longer available, no date given", "REMOVED_UNKNOWN", true},
		{"listing says no longer available, no date given", "ACTIVE", false},
	}
	for _, c := range cases {
		if got := agrees(c.say, c.status); got != c.want {
			t.Errorf("agrees(%q, %s) = %v, want %v", c.say, c.status, got, c.want)
		}
	}
}
