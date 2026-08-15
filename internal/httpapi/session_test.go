package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/gemini"
	"github.com/nazboyko/good-dog/internal/session"
)

// stubProvider serves one dog with a real photo file on disk.
type stubProvider struct {
	dog animal.Animal
	org animal.Organization
}

func (p stubProvider) Search(context.Context) ([]animal.Animal, error) {
	return []animal.Animal{p.dog}, nil
}
func (p stubProvider) GetAnimal(_ context.Context, id string) (*animal.Animal, error) {
	return &p.dog, nil
}
func (p stubProvider) GetOrganization(context.Context, string) (*animal.Organization, error) {
	return &p.org, nil
}
func (p stubProvider) GetStatus(context.Context, string) (animal.Status, error) {
	return animal.StatusActive, nil
}

// failingLLM makes the compiler serve the default sheet, which is
// exactly what a playtest without quota should still survive.
type failingLLM struct{}

func (failingLLM) GenerateJSON(context.Context, string, gemini.Options) ([]byte, error) {
	return nil, &gemini.APIError{Status: 503, Detail: "test"}
}

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	photo := filepath.Join(dir, "venus.jpg")
	if err := os.WriteFile(photo, []byte("not really a jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
	dog := animal.Animal{
		ID: "rsmn-a-9548", Name: "Venus", Breed: "Pit mix", AgeGroup: "Adult", Sex: "Female",
		Description: strings.Repeat("Goofy. Lap pet. ", 20), PhotoLocal: photo,
		ListingURL: "https://example.org/venus", OrgID: "org", LongStay: true,
		LongStayEvidence: animal.LongStayPlacement, RetrievedAt: time.Now(),
		Status: animal.StatusActive,
	}
	org := animal.Organization{ID: "org", Name: "Ruff Start Rescue", City: "Princeton", State: "MN"}
	compiler := dogsheet.NewCompiler(failingLLM{}, dogsheet.NewCache(filepath.Join(dir, "sheets")))
	h := NewSessions(stubProvider{dog: dog, org: org}, compiler, "rsmn-a-9548")
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, photo
}

func post(t *testing.T, url, body string) (*http.Response, session.View) {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var v session.View
	if res.StatusCode < 300 {
		json.NewDecoder(res.Body).Decode(&v)
	}
	res.Body.Close()
	return res, v
}

func TestSessionRunOnRails(t *testing.T) {
	srv, _ := newTestServer(t)

	res, v := post(t, srv.URL+"/api/session", "")
	if res.StatusCode != http.StatusCreated || v.Beat != session.BeatWake || v.Name != "Venus" {
		t.Fatalf("start: status %d view %+v", res.StatusCode, v)
	}
	base := srv.URL + "/api/session/" + v.SessionID

	// photo is forbidden long before the epilogue
	if r, _ := http.Get(base + "/photo"); r.StatusCode != http.StatusForbidden {
		t.Fatalf("photo before epilogue must be 403, got %d", r.StatusCode)
	}

	res, v = post(t, base+"/advance", "")
	if v.Beat != session.BeatScent || v.Scent == nil {
		t.Fatalf("scent beat: %+v", v)
	}
	res, v = post(t, base+"/advance", "")
	if v.Beat != session.BeatVisitor || v.Visitor == nil || len(v.Visitor.Options) != 6 {
		t.Fatalf("visitor beat: %+v", v)
	}
	// the visitor is waiting, no advancing past her
	if res, _ = post(t, base+"/advance", ""); res.StatusCode != http.StatusConflict {
		t.Fatalf("advance without a signal must 409, got %d", res.StatusCode)
	}
	if res, _ = post(t, base+"/vocalize", `{"signal":"meow"}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("bad signal must 409, got %d", res.StatusCode)
	}
	res, v = post(t, base+"/vocalize", `{"signal":"whine"}`)
	if v.Visitor == nil || v.Visitor.Mismatch == nil || v.Visitor.Mismatch.Meant == "" {
		t.Fatalf("vocalize must return the narrator: %+v", v)
	}
	res, v = post(t, base+"/advance", "")
	if v.Beat != session.BeatNight || v.Night == nil || len(v.Night.Story) < 4 {
		t.Fatalf("night beat: %+v", v)
	}
	res, v = post(t, base+"/advance", "")
	if v.Beat != session.BeatEpilogue || v.Epilogue == nil {
		t.Fatalf("epilogue beat: %+v", v)
	}
	e := v.Epilogue
	if e.ListingURL != "https://example.org/venus" || e.OrgName != "Ruff Start Rescue" || !e.LongStay {
		t.Errorf("epilogue facts: %+v", e)
	}
	// and now the photo route opens
	r, err := http.Get(srv.URL + e.PhotoURL)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != http.StatusOK {
		t.Fatalf("photo at epilogue must serve, got %d", r.StatusCode)
	}
	if r.Header.Get("Cache-Control") != "private, no-store" {
		t.Errorf("photo must not be cacheable, got %q", r.Header.Get("Cache-Control"))
	}
}

func TestSessionUnknownIDs(t *testing.T) {
	srv, _ := newTestServer(t)
	if r, _ := http.Get(srv.URL + "/api/session/nope"); r.StatusCode != http.StatusNotFound {
		t.Errorf("unknown session must 404, got %d", r.StatusCode)
	}
	if r, _ := post(t, srv.URL+"/api/session/nope/advance", ""); r.StatusCode != http.StatusNotFound {
		t.Errorf("advance on unknown session must 404, got %d", r.StatusCode)
	}
}

func TestPinnedDogMustBeRealAndActive(t *testing.T) {
	dir := t.TempDir()
	fake := animal.Animal{ID: "example-1", Name: "Biscuit", Synthetic: true, Status: animal.StatusActive}
	compiler := dogsheet.NewCompiler(nil, dogsheet.NewCache(filepath.Join(dir, "sheets")))
	h := NewSessions(stubProvider{dog: fake, org: animal.Organization{ID: "org"}}, compiler, "example-1")
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	if r, _ := post(t, srv.URL+"/api/session", ""); r.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("a pinned synthetic dog must never start a session, got %d", r.StatusCode)
	}
}

func TestSessionSurvivesDefaultSheet(t *testing.T) {
	// the failing llm means every session runs on the default sheet,
	// and the run must still reach the reveal
	srv, _ := newTestServer(t)
	_, v := post(t, srv.URL+"/api/session", "")
	if v.SessionID == "" {
		t.Fatal("a session must start even when the model is down")
	}
}
