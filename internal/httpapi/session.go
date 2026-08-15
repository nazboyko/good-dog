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

func NewSessions(provider animal.Provider, compiler *dogsheet.Compiler, firstDog string) *Sessions {
	return &Sessions{
		provider: provider,
		compiler: compiler,
		store:    session.NewStore(),
		firstDog: firstDog,
		now:      time.Now,
	}
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
	h.store.Put(s)
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
	s, ok := h.store.Get(r.PathValue("id"))
	if !ok {
		http.Error(w, "that life is not here", http.StatusNotFound)
		return nil, false
	}
	return s, true
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
