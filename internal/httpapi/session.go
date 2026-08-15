package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/session"
)

// Sessions wires the run endpoints. Every response is a session.View,
// so the client can only ever see what the current beat allows.
type Sessions struct {
	provider animal.Provider
	compiler *dogsheet.Compiler
	store    *session.Store
	// firstDog pins the playtest dog, empty means random from the pool
	firstDog string
	now      func() time.Time
}

// NewSessions wires the store over the DB. A nil db is memory only.
func NewSessions(provider animal.Provider, compiler *dogsheet.Compiler, db *session.DB, firstDog string) *Sessions {
	h := &Sessions{provider: provider, compiler: compiler, firstDog: firstDog, now: time.Now}
	h.store = session.NewStore(db, h.rebuild)
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
	if dog.Synthetic || dog.Status != animal.StatusActive {
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
	return session.Resume(row.ID, *dog, *org, sheet, rails, row.State), nil
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
	dog, err := h.pickDog(ctx)
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
	s := session.New(dog, *org, sheet, session.ShortRun, h.now())
	if err := h.store.Put(ctx, s, h.now()); err != nil {
		log.Printf("session start: save %s: %v", s.ID, err)
	}
	writeJSON(w, http.StatusCreated, s.View(h.now()))
}

// pickDog serves the pinned playtest dog when set, otherwise the first
// active dog in the pool. Random choice lands with the second dog.
func (h *Sessions) pickDog(ctx context.Context) (animal.Animal, error) {
	if h.firstDog != "" {
		a, err := h.provider.GetAnimal(ctx, h.firstDog)
		if err != nil {
			return animal.Animal{}, err
		}
		// the pin bypasses Search, so it must keep Search's promises
		if a.Synthetic || a.Status != animal.StatusActive {
			return animal.Animal{}, fmt.Errorf("pinned dog %s is not a real active dog", h.firstDog)
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
	return pool[0], nil
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
