package animal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var _ Provider = (*FixtureProvider)(nil)

const fixturePath = "../../fixtures/dogs.json"

func TestFixtureProviderLoadsExampleShape(t *testing.T) {
	p, err := NewFixtureProvider(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	pool, err := p.Search(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range pool {
		if a.Synthetic {
			t.Errorf("synthetic dog %s was served to a player", a.ID)
		}
	}

	got, err := p.GetAnimal(ctx, "example-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Biscuit" || got.Status != StatusActive || !got.Synthetic {
		t.Errorf("unexpected animal %+v", got)
	}
	if strings.ContainsAny(got.Description, "<>") {
		t.Errorf("description was not sanitized: %q", got.Description)
	}
	if ok, reason := PoolFilter(*got); !ok {
		t.Errorf("example entry must model a pool worthy dog, got: %s", reason)
	}
	if got.ListingURL == "" || got.PhotoLocal == "" || got.RetrievedAt.IsZero() {
		t.Error("example entry is missing listing url, local photo or retrieved_at")
	}
	if _, err := p.GetOrganization(ctx, got.OrgID); err != nil {
		t.Errorf("org lookup failed: %v", err)
	}
}

func TestFixtureProviderServesRealDogs(t *testing.T) {
	p := writeAndLoad(t, validFixtureJSON)
	pool, err := p.Search(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pool) != 1 || pool[0].Name != "Real Dog" {
		t.Fatalf("a real active dog must be served, pool: %+v", pool)
	}
}

func TestFixtureProviderStatusForMissingDog(t *testing.T) {
	p, err := NewFixtureProvider(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	status, err := p.GetStatus(context.Background(), "never-existed")
	if err != nil {
		t.Fatal(err)
	}
	if status != StatusRemovedUnknown {
		t.Errorf("missing dog status = %s, want REMOVED_UNKNOWN", status)
	}
}

func TestFixtureProviderRejectsBrokenFiles(t *testing.T) {
	cases := map[string]string{
		"bad version":      strings.Replace(validFixtureJSON, `"version": 1`, `"version": 2`, 1),
		"unknown status":   strings.Replace(validFixtureJSON, `"ACTIVE"`, `"NAPPING"`, 1),
		"unknown org":      strings.Replace(validFixtureJSON, `"org_id": "org-1"`, `"org_id": "ghost"`, 1),
		"no retrieved_at":  strings.Replace(validFixtureJSON, `, "retrieved_at": "2026-08-14T23:00:00Z"`, "", 1),
		"duplicate dog id": strings.Replace(validFixtureJSON, `"dogs": [`, `"dogs": [`+validDogJSON+",", 1),
		"long stay with no evidence": strings.Replace(validFixtureJSON,
			`"id": "real-1",`, `"id": "real-1", "long_stay": true,`, 1),
		"long stay with unknown evidence": strings.Replace(validFixtureJSON,
			`"id": "real-1",`, `"id": "real-1", "long_stay": true, "long_stay_evidence": "a hunch",`, 1),
		"evidence without the flag": strings.Replace(validFixtureJSON,
			`"id": "real-1",`, `"id": "real-1", "long_stay_evidence": "placement",`, 1),
	}
	for name, content := range cases {
		if _, err := newFixtureFromString(t, content); err == nil {
			t.Errorf("%s must fail to load", name)
		}
	}
}

const validDogJSON = `{
  "id": "real-1", "name": "Real Dog", "breed": "Mix", "age_group": "Adult",
  "sex": "Female", "size": "Medium",
  "description_html": "<p>` + "A very good dog who loves naps, snacks, gentle walks and every person she has ever met. Staff say she waits by the door each morning and settles in her bed each night without a fuss. She knows sit and shake." + `</p>",
  "photo_url": "https://example.org/p.jpg", "photo_local": "fixtures/photos/real-1.jpg",
  "listing_url": "https://example.org/adopt/real", "org_id": "org-1",
  "status": "ACTIVE", "retrieved_at": "2026-08-14T23:00:00Z"
}`

const validFixtureJSON = `{
  "version": 1,
  "organizations": [{"id": "org-1", "name": "Org", "city": "C", "state": "S", "url": "https://example.org"}],
  "dogs": [` + validDogJSON + `]
}`

func newFixtureFromString(t *testing.T, content string) (*FixtureProvider, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dogs.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewFixtureProvider(path)
}

func writeAndLoad(t *testing.T, content string) *FixtureProvider {
	t.Helper()
	p, err := newFixtureFromString(t, content)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
