package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
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
	"github.com/nazboyko/good-dog/internal/visitor"
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

// testWorld is everything a test needs to stand a server up, and to
// stand a second one up over the same database to simulate a restart.
type testWorld struct {
	provider stubProvider
	compiler *dogsheet.Compiler
	dbPath   string
	photo    string
}

func newTestWorld(t *testing.T) testWorld {
	t.Helper()
	dir := t.TempDir()
	photo := filepath.Join(dir, "venus.png")
	// a real 3x2 png so the epilogue can read the size from the header
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(photo, buf.Bytes(), 0o644); err != nil {
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
	return testWorld{provider: stubProvider{dog: dog, org: org}, compiler: compiler, dbPath: filepath.Join(dir, "sessions.db"), photo: photo}
}

// boot stands a server up over the world's database, like a process start.
func (w testWorld) boot(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := session.OpenDB(w.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	h := NewSessions(w.provider, w.compiler, db, session.ShortRun, "rsmn-a-9548")
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	w := newTestWorld(t)
	return w.boot(t), w.photo
}

// playExchanges answers n exchanges over http and returns the view the
// last one produced.
func playExchanges(t *testing.T, base, signal string, n int) session.View {
	t.Helper()
	var v session.View
	for i := 0; i < n; i++ {
		if res, got := post(t, base+"/vocalize", `{"signal":"`+signal+`"}`); res.StatusCode == http.StatusOK {
			v = got
		} else {
			t.Fatalf("exchange %d: vocalize got %d", i, res.StatusCode)
		}
		if res, got := post(t, base+"/advance", ""); res.StatusCode == http.StatusOK {
			v = got
		} else {
			t.Fatalf("exchange %d: advance got %d", i, res.StatusCode)
		}
	}
	return v
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
	if v.Visitor.Body == "" || v.Visitor.Arc != "" {
		t.Fatalf("the first answer reads a body but not the whole shape: %+v", v.Visitor)
	}
	// three more answers close the visit
	if res, _ = post(t, base+"/advance", ""); res.StatusCode != http.StatusOK {
		t.Fatalf("advance after the first answer got %d", res.StatusCode)
	}
	v = playExchanges(t, base, "whine", visitor.ExchangesPerScene-1)
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
	if e.PhotoWidth != 3 || e.PhotoHeight != 2 {
		t.Errorf("epilogue must carry the photo size for the reserved box, got %dx%d", e.PhotoWidth, e.PhotoHeight)
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
	h := NewSessions(stubProvider{dog: fake, org: animal.Organization{ID: "org"}}, compiler, nil, session.ShortRun, "example-1")
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

func TestSessionResumesAcrossAServerRestart(t *testing.T) {
	w := newTestWorld(t)
	first := w.boot(t)

	// play to the visitor and answer, every step snapshots to the db
	_, v := post(t, first.URL+"/api/session", "")
	base := "/api/session/" + v.SessionID
	post(t, first.URL+base+"/advance", "")
	post(t, first.URL+base+"/advance", "")
	_, before := post(t, first.URL+base+"/vocalize", `{"signal":"whine"}`)
	first.Close()

	// a new process over the same database, the cache is empty
	second := w.boot(t)
	res, err := http.Get(second.URL + base)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("resume after restart must find the life, got %d", res.StatusCode)
	}
	var after session.View
	json.NewDecoder(res.Body).Decode(&after)
	res.Body.Close()
	if after.Beat != session.BeatVisitor || after.Visitor == nil || after.Visitor.Signal != "whine" {
		t.Fatalf("resumed at the wrong place: %+v", after)
	}
	if after.Visitor.Mismatch == nil || after.Visitor.Mismatch.Heard != before.Visitor.Mismatch.Heard {
		t.Errorf("the narrator must read the same after restart")
	}
	// and it keeps playing the same visit from exactly there
	_, next := post(t, second.URL+base+"/advance", "")
	if next.Beat != session.BeatVisitor || next.Visitor.Exchange != 2 {
		t.Errorf("a resume mid visit continues the visit: %s exchange %d", next.Beat, next.Visitor.Exchange)
	}
	// the answer given before the restart was banked by that advance,
	// so one is in the scene and three remain
	next = playExchanges(t, second.URL+base, "whine", visitor.ExchangesPerScene-1)
	if next.Beat != session.BeatNight {
		t.Errorf("finishing the resumed visit moves the day on: %s", next.Beat)
	}
}

func TestSessionUnknownOrExpiredIDIsACleanMiss(t *testing.T) {
	w := newTestWorld(t)
	srv := w.boot(t)
	// unknown id: 404 with the stable body, never a 500
	res, _ := http.Get(srv.URL + "/api/session/never-existed")
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("unknown id must be 404, got %d", res.StatusCode)
	}
	// an expired session: written, purged, then asked for
	_, v := post(t, srv.URL+"/api/session", "")
	db, err := session.OpenDB(w.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Purge(context.Background(), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	db.Close()
	// still cached in this process, so it resumes; a restart makes it a miss
	srv.Close()
	again := w.boot(t)
	res, _ = http.Get(again.URL + "/api/session/" + v.SessionID)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("an expired session after restart must be a clean 404, got %d", res.StatusCode)
	}
	// and a fresh start still works, the player gets a new life
	if r, nv := post(t, again.URL+"/api/session", ""); r.StatusCode != http.StatusCreated || nv.Beat != session.BeatWake {
		t.Errorf("a clean new run must start after a miss: %d %+v", r.StatusCode, nv)
	}
}

// poolProvider is a shelter with several real dogs in it, which is what
// selection needs and the single dog stub cannot give.
type poolProvider struct {
	dogs []animal.Animal
	org  animal.Organization
}

func (p poolProvider) Search(context.Context) ([]animal.Animal, error) { return p.dogs, nil }
func (p poolProvider) GetAnimal(_ context.Context, id string) (*animal.Animal, error) {
	for i := range p.dogs {
		if p.dogs[i].ID == id {
			return &p.dogs[i], nil
		}
	}
	return nil, errors.New("no such dog")
}
func (p poolProvider) GetOrganization(context.Context, string) (*animal.Organization, error) {
	return &p.org, nil
}
func (p poolProvider) GetStatus(context.Context, string) (animal.Status, error) {
	return animal.StatusActive, nil
}

func newPool(t *testing.T, n int) (*Sessions, poolProvider) {
	t.Helper()
	org := animal.Organization{ID: "org", Name: "Ruff Start Rescue", City: "Princeton", State: "MN"}
	var dogs []animal.Animal
	for i := 0; i < n; i++ {
		dogs = append(dogs, animal.Animal{
			ID: fmt.Sprintf("dog-%d", i), Name: fmt.Sprintf("Dog %d", i), Breed: "mix",
			AgeGroup: "Adult", Sex: "Female", Description: strings.Repeat("Good. ", 20),
			ListingURL: "https://example.org/x", OrgID: "org", RetrievedAt: time.Now(),
			Status: animal.StatusActive,
		})
	}
	prov := poolProvider{dogs: dogs, org: org}
	compiler := dogsheet.NewCompiler(failingLLM{}, dogsheet.NewCache(t.TempDir()))
	h := NewSessions(prov, compiler, nil, session.ShortRun, "")
	return h, prov
}

// Every run is somebody else's life. A game that always opens on the
// same dog turns that dog into a mascot, which is the opposite of the
// point, so selection is spread across the pool.
func TestARunPicksFromTheWholePool(t *testing.T) {
	h, prov := newPool(t, 5)
	seen := map[string]bool{}
	for i := range prov.dogs {
		h.choose = func(int) int { return i }
		dog, err := h.pickDog(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		seen[dog.ID] = true
	}
	if len(seen) != len(prov.dogs) {
		t.Errorf("only %d of %d dogs in the pool can ever be picked: %v", len(seen), len(prov.dogs), seen)
	}
}

// Living another life hands you somebody new: the dog just finished is
// excluded, so the same animal never comes back to back.
func TestAnotherLifeIsNeverTheSameDogTwiceRunning(t *testing.T) {
	h, prov := newPool(t, 5)
	previous := prov.dogs[0].ID
	seen := map[string]bool{}
	for i := 0; i < len(prov.dogs)-1; i++ {
		h.choose = func(int) int { return i }
		dog, err := h.pickDog(context.Background(), []string{previous})
		if err != nil {
			t.Fatal(err)
		}
		if dog.ID == previous {
			t.Fatalf("chooser %d handed back the dog just played: %s", i, dog.ID)
		}
		seen[dog.ID] = true
	}
	// excluding one must not collapse the pool to a single alternative
	if len(seen) != len(prov.dogs)-1 {
		t.Errorf("excluding one dog left %d reachable, want %d", len(seen), len(prov.dogs)-1)
	}
}

// A pool that has shrunk to the dog you just played gives you that dog
// again rather than refusing to start.
func TestALastDogStandingIsStillALife(t *testing.T) {
	h, prov := newPool(t, 1)
	h.choose = func(int) int { return 0 }
	dog, err := h.pickDog(context.Background(), []string{prov.dogs[0].ID})
	if err != nil {
		t.Fatalf("a single dog pool must still start a run: %v", err)
	}
	if dog.ID != prov.dogs[0].ID {
		t.Errorf("got %s, the only dog there is is %s", dog.ID, prov.dogs[0].ID)
	}
}

// The pin is for playtesting one dog on purpose and still wins.
func TestThePinBeatsSelection(t *testing.T) {
	h, prov := newPool(t, 5)
	h.firstDog = prov.dogs[1].ID
	h.choose = func(int) int { return 0 }
	dog, err := h.pickDog(context.Background(), []string{prov.dogs[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	if dog.ID != prov.dogs[1].ID {
		t.Errorf("the pin must win even against the exclusion, got %s", dog.ID)
	}
}

// Excluding only the last dog still let a three run sitting circle two
// animals while ten others waited. The recent few are all excluded.
func TestRecentDogsAreAllSkipped(t *testing.T) {
	h, prov := newPool(t, 5)
	recent := []string{prov.dogs[0].ID, prov.dogs[1].ID, prov.dogs[2].ID}
	seen := map[string]bool{}
	for i := 0; i < len(prov.dogs)-len(recent); i++ {
		h.choose = func(int) int { return i }
		dog, err := h.pickDog(context.Background(), recent)
		if err != nil {
			t.Fatal(err)
		}
		for _, skipped := range recent {
			if dog.ID == skipped {
				t.Errorf("chooser %d handed back a dog just played: %s", i, dog.ID)
			}
		}
		seen[dog.ID] = true
	}
	if len(seen) != len(prov.dogs)-len(recent) {
		t.Errorf("two dogs should have been left, %d were reachable", len(seen))
	}
}

// When every dog is a recent one the run still starts.
func TestAPoolOfNothingButRecentDogsStillStarts(t *testing.T) {
	h, prov := newPool(t, 3)
	all := []string{prov.dogs[0].ID, prov.dogs[1].ID, prov.dogs[2].ID}
	h.choose = func(n int) int {
		if n != len(prov.dogs) {
			t.Errorf("the fallback should offer the whole pool, got %d", n)
		}
		return 0
	}
	if _, err := h.pickDog(context.Background(), all); err != nil {
		t.Fatalf("a fully recent pool must still start a run: %v", err)
	}
}

// Real shelters are full of dogs called Bella, and three of the twelve
// curated ones are. Two runs in a row opening on that name reads as a
// stuck picker even when they are two different animals, so the recent
// exclusion matches on name as well as on id.
func TestARecentNameIsSkippedEvenOnADifferentDog(t *testing.T) {
	h, prov := newPool(t, 4)
	// two different dogs, same name, which is the real fixture problem
	prov.dogs[1].Name = prov.dogs[0].Name
	h.provider = prov

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		h.choose = func(int) int { return i }
		dog, err := h.pickDog(context.Background(), []string{prov.dogs[0].ID})
		if err != nil {
			t.Fatal(err)
		}
		if dog.Name == prov.dogs[0].Name {
			t.Errorf("chooser %d handed back the name just played: %s (%s)", i, dog.Name, dog.ID)
		}
		seen[dog.ID] = true
	}
	if len(seen) != 2 {
		t.Errorf("two dogs should have been left after skipping a shared name, got %d", len(seen))
	}
}

// The whole path for a dog who went home while the game was being
// built. Nothing here calls RealityLine directly: the session starts
// over http, walks the rails, and the assertion is on the payload the
// browser would render. That is the only place the promise is kept.
func TestAnAdoptedDogPlaysAndTellsTheTruth(t *testing.T) {
	w := newTestWorld(t)
	dog := w.provider.dog
	dog.Status = animal.StatusAdoptedConfirmed
	dog.AdoptedOn = "August 15, 2026"
	w.provider = stubProvider{dog: dog, org: w.provider.org}
	srv := w.boot(t)

	res, v := post(t, srv.URL+"/api/session", "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("an adopted dog must still be playable, session start got %d", res.StatusCode)
	}
	base := srv.URL + "/api/session/" + v.SessionID
	post(t, base+"/advance", "") // scent
	post(t, base+"/advance", "") // visitor
	post(t, base+"/vocalize", `{"signal":"whine"}`)
	post(t, base+"/advance", "")
	v = playExchanges(t, base, "whine", visitor.ExchangesPerScene-1)
	if v.Beat != session.BeatNight || v.Night == nil {
		t.Fatalf("expected the night, got %+v", v)
	}
	// the night's own story about her follows her listing
	night := strings.Join(v.Night.Story, " ")
	if strings.Contains(night, "still here") || strings.Contains(night, "waiting") {
		t.Errorf("her own night says she is still here: %q", night)
	}
	if !strings.Contains(night, "went home") {
		t.Errorf("her own night never says she went home: %q", night)
	}

	_, v = post(t, base+"/advance", "")
	if v.Beat != session.BeatEpilogue || v.Epilogue == nil {
		t.Fatalf("epilogue beat: %+v", v)
	}
	e := v.Epilogue
	if !strings.Contains(e.RealityLine, "was adopted on August 15, 2026") {
		t.Errorf("the reveal does not tell the truth: %q", e.RealityLine)
	}
	if strings.Contains(e.RealityLine, "waiting") {
		t.Errorf("the reveal calls an adopted dog waiting: %q", e.RealityLine)
	}
	if !e.Seam {
		t.Error("an adopted dog's reveal must show the seam, the fiction and the listing disagree on every ending")
	}
	if !e.Adopted {
		t.Error("the client is not told she is adopted, so the button still asks to meet her")
	}
}

// The other direction: a dog gone from the listings with no stated
// reason must never produce adoption language anywhere the client can
// see. The whole payload is searched, not one field, because a new
// field added later would otherwise be a new place to leak.
func TestAGoneDogNeverProducesAdoptionLanguageAnywhere(t *testing.T) {
	w := newTestWorld(t)
	dog := w.provider.dog
	dog.Status = animal.StatusRemovedUnknown
	w.provider = stubProvider{dog: dog, org: w.provider.org}
	srv := w.boot(t)

	res, v := post(t, srv.URL+"/api/session", "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("session start got %d", res.StatusCode)
	}
	base := srv.URL + "/api/session/" + v.SessionID
	// every string value in the payload, so a field named adopted that
	// carries false is not mistaken for the word being said to a player
	var payloads []string
	record := func(v session.View) {
		raw, _ := json.Marshal(v)
		var any interface{}
		json.Unmarshal(raw, &any)
		payloads = append(payloads, strings.Join(stringValues(any), " | "))
	}
	record(v)
	_, v = post(t, base+"/advance", "")
	record(v)
	_, v = post(t, base+"/advance", "")
	record(v)
	_, v = post(t, base+"/vocalize", `{"signal":"whine"}`)
	record(v)
	post(t, base+"/advance", "")
	v = playExchanges(t, base, "whine", visitor.ExchangesPerScene-1)
	record(v)
	_, v = post(t, base+"/advance", "")
	record(v)
	if v.Beat != session.BeatEpilogue {
		t.Fatalf("never reached the epilogue: %s", v.Beat)
	}

	forbidden := []string{"adopt", "went home", "found a home", "forever home"}
	for i, p := range payloads {
		low := strings.ToLower(p)
		for _, word := range forbidden {
			if strings.Contains(low, word) {
				t.Errorf("payload %d for a REMOVED_UNKNOWN dog contains %q", i, word)
			}
		}
	}
	// and the reveal still says the careful thing
	if !strings.Contains(v.Epilogue.RealityLine, "no longer listed") {
		t.Errorf("careful wording missing: %q", v.Epilogue.RealityLine)
	}
}

// stringValues walks a decoded json value and returns every string in it,
// which is everything a player could read, and none of the field names.
func stringValues(v interface{}) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []interface{}:
		var out []string
		for _, e := range x {
			out = append(out, stringValues(e)...)
		}
		return out
	case map[string]interface{}:
		var out []string
		for _, e := range x {
			out = append(out, stringValues(e)...)
		}
		return out
	}
	return nil
}

// The three gates a dog has to pass to be played: the pool, the pin,
// and the resume after a restart. Each used to say ACTIVE only, which
// deleted an adopted dog from the game. All three go through the real
// FixtureProvider here so a filter put back in any one of them fails.
func TestAnAdoptedDogPassesEveryGate(t *testing.T) {
	dir := t.TempDir()
	// the smallest real fixture file: one dog, adopted, and her org
	fixture := `{
	  "version": 1,
	  "organizations": [{"id": "org", "name": "Ruff Start Rescue", "city": "Princeton", "state": "MN"}],
	  "dogs": [{
	    "id": "rsmn-a-1", "name": "Bella", "breed": "Pit mix", "age_group": "Adult", "sex": "Female",
	    "description_html": "<p>` + strings.Repeat("Goofy. Lap pet. ", 20) + `</p>",
	    "listing_url": "https://example.org/bella", "org_id": "org",
	    "photo_url": "https://example.org/bella.jpg",
	    "status": "ADOPTED_CONFIRMED", "adopted_on": "August 15, 2026",
	    "retrieved_at": "2026-08-15T00:00:00Z"
	  }]
	}`
	path := filepath.Join(dir, "dogs.json")
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	provider, err := animal.NewFixtureProvider(path)
	if err != nil {
		t.Fatal(err)
	}
	compiler := dogsheet.NewCompiler(failingLLM{}, dogsheet.NewCache(filepath.Join(dir, "sheets")))
	db, err := session.OpenDB(filepath.Join(dir, "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// gate one, the pool: no pin, so she has to come out of Search
	boot := func(pin string) *httptest.Server {
		h := NewSessions(provider, compiler, db, session.ShortRun, pin)
		mux := http.NewServeMux()
		h.Register(mux)
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		return srv
	}
	pool := boot("")
	res, v := post(t, pool.URL+"/api/session", "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("the pool refused an adopted dog: %d", res.StatusCode)
	}
	if v.Name != "Bella" {
		t.Fatalf("the pool played %q, want Bella", v.Name)
	}

	// gate two, the pin
	pinned := boot("rsmn-a-1")
	if res, _ := post(t, pinned.URL+"/api/session", ""); res.StatusCode != http.StatusCreated {
		t.Fatalf("the pin refused an adopted dog: %d", res.StatusCode)
	}

	// gate three, the resume: a fresh process over the same database
	base := "/api/session/" + v.SessionID
	pool.Close()
	again := boot("")
	r, err := http.Get(again.URL + base)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("a life as an adopted dog was thrown away on restart: %d", r.StatusCode)
	}
}
