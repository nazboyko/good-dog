package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/radio"
	"github.com/nazboyko/good-dog/internal/session"
)

// Sessions wires the run endpoints. Every response is a session.View,
// so the client can only ever see what the current beat allows.
type Sessions struct {
	provider animal.Provider
	compiler *dogsheet.Compiler
	// the host's recordings, read only at play time
	voice radio.VoiceCache
	store *session.Store
	// rails is which run new sessions start on, short for the playtest
	rails session.Rails
	// firstDog pins the playtest dog, empty means random from the pool
	firstDog string
	// choose picks an index below n. Swappable so a test can pin the
	// choice rather than reaching for a seed.
	choose func(n int) int
	now    func() time.Time
}

// NewSessions wires the store over the DB. A nil db is memory only.
func NewSessions(provider animal.Provider, compiler *dogsheet.Compiler, db *session.DB, rails session.Rails, firstDog string) *Sessions {
	h := &Sessions{provider: provider, compiler: compiler, rails: rails, firstDog: firstDog, now: time.Now,
		choose: rand.IntN}
	h.store = session.NewStore(db, h.rebuild)
	return h
}

// Unvoiced walks every night the pool can produce, each dog as the one
// the player lives as and as a neighbour, and returns every line with
// no recording behind it. Boot logs the result and never records: a
// missing line has to be a line in the log, not silence at two in the
// morning on the emotional close of somebody's night.
func (h *Sessions) Unvoiced(ctx context.Context) []string {
	if h.voice == nil {
		return nil
	}
	pool, err := h.provider.Search(ctx)
	if err != nil {
		// not a missing recording, so not the make voices advice: say
		// what it is
		log.Printf("radio: could not read the pool to check recordings: %v", err)
		return nil
	}
	seen := map[string]bool{}
	var out []string
	note := func(dog string, cues []radio.Cue) {
		_, missing := radio.WithVoices(cues, h.voice)
		for _, m := range missing {
			key := dog + ": " + m
			if !seen[key] {
				seen[key] = true
				out = append(out, key)
			}
		}
	}
	for _, dog := range pool {
		org, err := h.provider.GetOrganization(ctx, dog.OrgID)
		if err != nil {
			continue
		}
		sheet, err := h.compiler.Compile(ctx, dog)
		var degraded *dogsheet.Degraded
		if err != nil && !errors.As(err, &degraded) {
			continue
		}
		voice := radio.VoiceFor(dog, sheet)
		note(dog.Name, radio.Broadcast(nil, session.RadioStory(dog, sheet), voice, 0))
		note(dog.Name, radio.Broadcast([]radio.Neighbour{{Dog: dog, Org: *org, Sheet: sheet}}, nil, voice, 0))
	}
	return out
}

// WithVoice gives the host his recordings. Optional: without it the
// night still plays, in text, which is the same night a player with the
// sound off gets anyway.
func (h *Sessions) WithVoice(vc radio.VoiceCache) *Sessions {
	h.voice = vc
	return h
}

// rebuild turns a persisted row back into a live session: the dog and
// org from the provider, the sheet through the compiler's cache. If the
// dog has left the pool since, the row is a miss and the player gets a
// clean new run.
func (h *Sessions) rebuild(ctx context.Context, row session.Row) (*session.Session, error) {
	rails, ok := session.RailsByName(row.Rails)
	if !ok {
		return nil, fmt.Errorf("unknown rails %q", row.Rails)
	}
	dog, err := h.provider.GetAnimal(ctx, row.DogID)
	if err != nil {
		return nil, err
	}
	if dog.Synthetic || !animal.Playable(dog.Status) {
		return nil, fmt.Errorf("dog %s is no longer playable", row.DogID)
	}
	org, err := h.provider.GetOrganization(ctx, row.OrgID)
	if err != nil {
		return nil, err
	}
	sheet, err := h.compiler.Compile(ctx, *dog)
	var degraded *dogsheet.Degraded
	if err != nil && !errors.As(err, &degraded) {
		return nil, err
	}
	s := session.Resume(row.ID, *dog, *org, sheet, rails, row.State)
	s.Tonight(h.tonight(ctx, *dog, session.RadioStory(*dog, sheet), radio.VoiceFor(*dog, sheet), nightSeed(row.ID)))
	return s, nil
}

// tonight builds the night broadcast from other real dogs in the pool.
// Sheets come from the compiler's cache, so this costs no model calls
// on a warm cache. A night with no neighbours is a quiet night, never
// an error: the run matters more than the radio.
// tonight builds the night. seed picks which of Ranger's lines this one
// gets, and it has to be stable for the session: the stream and the
// client's fallback both build the night independently, and a seed that
// moved between them would play two different broadcasts.
func (h *Sessions) tonight(ctx context.Context, playing animal.Animal, own []string, ownVoice radio.Voice, seed int64) []radio.Cue {
	pool, err := h.provider.Search(ctx)
	if err != nil {
		log.Printf("radio: pool unavailable, tonight is quiet: %v", err)
		return nil
	}
	var candidates []radio.Neighbour
	for _, dog := range pool {
		if dog.ID == playing.ID || dog.OrgID == playing.OrgID {
			continue
		}
		org, err := h.provider.GetOrganization(ctx, dog.OrgID)
		if err != nil {
			continue
		}
		sheet, err := h.compiler.Compile(ctx, dog)
		var degraded *dogsheet.Degraded
		if err != nil && !errors.As(err, &degraded) {
			continue
		}
		candidates = append(candidates, radio.Neighbour{Dog: dog, Org: *org, Sheet: sheet})
	}
	// no cap here: Neighbours still has to drop duplicate names and
	// default sheets, and capping the candidates first meant one
	// duplicate inside the first three left the night a story short
	cues, missing := radio.WithVoices(
		radio.Broadcast(radio.Neighbours(candidates, playing, playing.OrgID), own, ownVoice, seed), h.voice)
	if len(missing) > 0 {
		// prepared at boot, so a miss here means the cache moved under us
		log.Printf("radio: %d host lines have no recording, tonight is quieter than it should be: %v",
			len(missing), missing)
	}
	return cues
}

// Tonight is the Nightly side of the stream: what this session's night
// holds. An unknown or finished session gets nothing, which the stream
// serves as a connection with no cues rather than an error.
func (h *Sessions) Tonight(ctx context.Context, sessionID string) []radio.Cue {
	if sessionID == "" {
		return nil
	}
	s, err := h.store.Get(ctx, sessionID)
	if err != nil {
		return nil
	}
	v := s.View(h.now())
	if v.Night == nil {
		return nil
	}
	return v.Night.Radio
}

// nightSeed turns a session id into a stable number, so the same life
// always hears the same night however many times it reconnects.
func nightSeed(id string) int64 {
	var n int64
	for _, r := range id {
		n = n*31 + int64(r)
	}
	if n < 0 {
		n = -n
	}
	return n
}

func (h *Sessions) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/session", h.start)
	mux.HandleFunc("GET /api/session/{id}", h.view)
	mux.HandleFunc("POST /api/session/{id}/vocalize", h.vocalize)
	mux.HandleFunc("POST /api/session/{id}/advance", h.advance)
	mux.HandleFunc("GET /api/session/{id}/photo", h.photo)
}

func (h *Sessions) start(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var start struct {
		// the sessions this player has just finished, newest first, so
		// the next life is somebody they have not just been. Absent on
		// a first run.
		After []string `json:"after"`
	}
	if r.Body != nil {
		// a missing or unreadable body is a plain first run, never an error
		_ = json.NewDecoder(r.Body).Decode(&start)
	}
	dog, err := h.pickDog(ctx, h.dogsBehind(ctx, start.After))
	if err != nil {
		log.Printf("session start: %v", err)
		http.Error(w, "no dog is available right now", http.StatusServiceUnavailable)
		return
	}
	org, err := h.provider.GetOrganization(ctx, dog.OrgID)
	if err != nil {
		http.Error(w, "the shelter record is missing", http.StatusServiceUnavailable)
		return
	}
	sheet, err := h.compiler.Compile(ctx, dog)
	var degraded *dogsheet.Degraded
	if err != nil && !errors.As(err, &degraded) {
		log.Printf("session start: compile %s: %v", dog.ID, err)
		http.Error(w, "could not wake up right now", http.StatusServiceUnavailable)
		return
	}
	s := session.New(dog, *org, sheet, h.rails, h.now())
	s.Tonight(h.tonight(ctx, dog, session.RadioStory(dog, sheet), radio.VoiceFor(dog, sheet), nightSeed(s.ID)))
	if err := h.store.Put(ctx, s, h.now()); err != nil {
		log.Printf("session start: save %s: %v", s.ID, err)
	}
	writeJSON(w, http.StatusCreated, s.View(h.now()))
}

// pickDog chooses whose life this run is.
//
// The pin is for playtesting one dog on purpose. Without it, a run gets
// a real dog at random from the pool, because the whole point is that
// the life belongs to a different animal each time and a game that
// always opens on the same dog quietly turns that dog into a mascot.
//
// recent is the handful of dogs this player has just been. All of them
// are excluded, so living another life never hands back the same animal
// twice running and does not circle a couple of dogs while nine others
// wait. If every dog left is a recent one, the oldest of them comes
// round again rather than the player getting nothing.
func (h *Sessions) pickDog(ctx context.Context, recent []string) (animal.Animal, error) {
	if h.firstDog != "" {
		a, err := h.provider.GetAnimal(ctx, h.firstDog)
		if err != nil {
			return animal.Animal{}, err
		}
		// the pin bypasses Search, so it must keep Search's promises
		if a.Synthetic || !animal.Playable(a.Status) {
			return animal.Animal{}, fmt.Errorf("pinned dog %s is not a real dog the game can play", h.firstDog)
		}
		return *a, nil
	}
	pool, err := h.provider.Search(ctx)
	if err != nil {
		return animal.Animal{}, err
	}
	if len(pool) == 0 {
		return animal.Animal{}, errors.New("empty pool")
	}
	// Skip by name as well as by id. Three of the twelve curated dogs
	// are called Bella, which is true of real shelters and reads as a
	// bug: two runs in a row opening on "Bella" looks like the picker
	// is stuck even when they are two different animals with different
	// breeds and different listings. A player identifies a dog by name.
	skip := make(map[string]bool, len(recent)*2)
	for _, id := range recent {
		skip[id] = true
		for _, a := range pool {
			if a.ID == id {
				skip[strings.ToLower(a.Name)] = true
			}
		}
	}
	fresh := make([]animal.Animal, 0, len(pool))
	for _, a := range pool {
		if !skip[a.ID] && !skip[strings.ToLower(a.Name)] {
			fresh = append(fresh, a)
		}
	}
	if len(fresh) == 0 {
		fresh = pool
	}
	return fresh[h.choose(len(fresh))], nil
}

// dogsBehind maps recently finished sessions to the dogs they were, so
// the next run can avoid them. Sessions the store has forgotten simply
// stop excluding anything, which is the right way for this to decay.
func (h *Sessions) dogsBehind(ctx context.Context, sessionIDs []string) []string {
	var out []string
	for _, id := range sessionIDs {
		if id == "" {
			continue
		}
		s, err := h.store.Get(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, s.DogID())
	}
	return out
}

func (h *Sessions) load(w http.ResponseWriter, r *http.Request) (*session.Session, bool) {
	s, err := h.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		// a bare ErrNotFound is a plain miss, anything else carries a
		// cause worth a log line: a dog gone from the pool, an unreadable row
		if err != session.ErrNotFound {
			log.Printf("session %s: %v", r.PathValue("id"), err)
		}
		// unknown, expired or unreadable: the client starts a clean run
		http.Error(w, "that life is not here", http.StatusNotFound)
		return nil, false
	}
	return s, true
}

// snapshot persists after a transition. A failed save is logged, never
// shown, the in memory session is still the live truth for this process.
// The request context is not used, a client hanging up mid transition
// must not skip the write.
func (h *Sessions) snapshot(ctx context.Context, s *session.Session) {
	if err := h.store.Put(context.WithoutCancel(ctx), s, h.now()); err != nil {
		log.Printf("session %s: snapshot: %v", s.ID, err)
	}
}

func (h *Sessions) view(w http.ResponseWriter, r *http.Request) {
	if s, ok := h.load(w, r); ok {
		writeJSON(w, http.StatusOK, s.View(h.now()))
	}
}

func (h *Sessions) vocalize(w http.ResponseWriter, r *http.Request) {
	s, ok := h.load(w, r)
	if !ok {
		return
	}
	var body struct {
		Signal session.Vocalization `json:"signal"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		http.Error(w, "bad signal", http.StatusBadRequest)
		return
	}
	if err := s.Vocalize(body.Signal); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	h.snapshot(r.Context(), s)
	writeJSON(w, http.StatusOK, s.View(h.now()))
}

func (h *Sessions) advance(w http.ResponseWriter, r *http.Request) {
	s, ok := h.load(w, r)
	if !ok {
		return
	}
	if err := s.Advance(); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	h.snapshot(r.Context(), s)
	writeJSON(w, http.StatusOK, s.View(h.now()))
}

// photo serves the real photo only once the session has reached the
// epilogue. Before that the file might as well not exist.
func (h *Sessions) photo(w http.ResponseWriter, r *http.Request) {
	s, ok := h.load(w, r)
	if !ok {
		return
	}
	if beat := s.Beat(); beat != session.BeatEpilogue && beat != session.BeatDone {
		http.Error(w, "not yet", http.StatusForbidden)
		return
	}
	f, err := os.Open(s.PhotoLocal())
	if err != nil {
		http.Error(w, "photo missing", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, "photo", h.now(), f)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
