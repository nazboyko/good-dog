package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/httpapi"
	"github.com/nazboyko/good-dog/internal/radio"
	"github.com/nazboyko/good-dog/internal/session"
)

// Two dogs, one library voice, two stabilities. That is the exact shape
// that left the close of a night unrecorded: the recorder deduped on the
// voice, so the second dog's read of the same sentence was never planned
// and the cache reported warm. Built here on purpose rather than read
// off the local sheet cache, which CI does not have, so the guard fires
// wherever it runs and not only on the machine that already knew.
func collidingPool(t *testing.T) (animal.Provider, *dogsheet.Compiler, []animal.Animal) {
	t.Helper()
	dir := t.TempDir()
	// same size and age, so VoiceFor picks the same kennel voice for both
	fixture := `{
	  "version": 1,
	  "organizations": [{"id": "org", "name": "Ruff Start Rescue", "city": "Princeton", "state": "MN"}],
	  "dogs": [
	    {"id": "a", "name": "Quiet", "sex": "Female", "age_group": "Adult", "weight_text": "20 lbs",
	     "description_html": "<p>` + strings.Repeat("Calm. ", 40) + `</p>", "photo_url": "https://x/a.jpg",
	     "listing_url": "https://x/a", "org_id": "org", "status": "ACTIVE", "retrieved_at": "2026-08-15T00:00:00Z"},
	    {"id": "b", "name": "Loud", "sex": "Female", "age_group": "Adult", "weight_text": "20 lbs",
	     "description_html": "<p>` + strings.Repeat("Barks. ", 40) + `</p>", "photo_url": "https://x/b.jpg",
	     "listing_url": "https://x/b", "org_id": "org", "status": "ACTIVE", "retrieved_at": "2026-08-15T00:00:00Z"}
	  ]
	}`
	path := filepath.Join(dir, "dogs.json")
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	provider, err := animal.NewFixtureProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := provider.Search(context.Background())
	if err != nil || len(pool) != 2 {
		t.Fatalf("pool: %v %d", err, len(pool))
	}

	// two sheets with the two voice profiles that pull stability apart,
	// written into a fresh disk cache the compiler will read back
	sheets := dogsheet.NewCache(filepath.Join(dir, "sheets"))
	for _, dog := range pool {
		profile := "Quiet and steady."
		if dog.ID == "b" {
			profile = "Loud and vocal."
		}
		sheet := &dogsheet.DogSheet{
			// the cache refuses a sheet whose id does not match the dog,
			// which is its own guard against serving one dog another's
			AnimalID:  dog.ID,
			Voice:     dogsheet.NarrativeInference{Value: profile},
			RadioSeed: dogsheet.NarrativeInference{Value: "Meet " + dog.Name + "."},
			Quirks:    []dogsheet.NarrativeInference{{Value: "Tilts her head at the radio."}},
		}
		if err := sheets.Put(dog, sheet); err != nil {
			t.Fatal(err)
		}
	}
	compiler := dogsheet.NewCompiler(nil, sheets)

	// prove the setup is the collision it claims to be, or the test
	// below proves nothing
	sa, _ := compiler.Compile(context.Background(), pool[0])
	sb, _ := compiler.Compile(context.Background(), pool[1])
	va, vb := radio.VoiceFor(pool[0], sa), radio.VoiceFor(pool[1], sb)
	if va.ID != vb.ID {
		t.Fatalf("setup: the two dogs must share a voice, got %s and %s", va.Name, vb.Name)
	}
	if va.Stability == vb.Stability {
		t.Fatalf("setup: the two dogs must differ in stability, both %.2f", va.Stability)
	}
	return provider, compiler, pool
}

// Every cue the night looks up has to be in the plan under the exact key
// the night uses. The plan and the night build cues the same way; what
// is independent here is the key, which is the thing that was wrong.
func TestThePlanCoversEveryLineTheNightLooksUp(t *testing.T) {
	provider, compiler, pool := collidingPool(t)
	ctx := context.Background()
	work, err := plan(ctx, provider, compiler, pool)
	if err != nil {
		t.Fatal(err)
	}
	planned := map[string]bool{}
	for _, l := range work {
		planned[httpapi.KeyFor(l.Text, l.Voice)] = true
	}

	for _, dog := range pool {
		sheet, err := compiler.Compile(ctx, dog)
		if err != nil && sheet == nil {
			t.Fatal(err)
		}
		voice := radio.VoiceFor(dog, sheet)
		org, err := provider.GetOrganization(ctx, dog.OrgID)
		if err != nil {
			t.Fatal(err)
		}
		// as the dog the player lives as, which is where the close is
		for _, c := range radio.Broadcast(nil, session.RadioStory(dog, sheet), voice, 0) {
			if !planned[httpapi.KeyFor(c.Line, c.Voice)] {
				t.Errorf("%s: the night looks up %q as %s at stability %.2f and the plan never records it",
					dog.Name, c.Line, c.Voice.Name, c.Voice.Stability)
			}
		}
		// and as a neighbour
		n := radio.Neighbour{Dog: dog, Org: *org, Sheet: sheet}
		for _, c := range radio.Broadcast([]radio.Neighbour{n}, nil, voice, 0) {
			if !planned[httpapi.KeyFor(c.Line, c.Voice)] {
				t.Errorf("%s as a neighbour: %q at %s/%.2f is not in the plan", dog.Name, c.Line, c.Voice.Name, c.Voice.Stability)
			}
		}
	}
}
