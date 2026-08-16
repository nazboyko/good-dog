package httpapi

import (
	"net/http"

	"github.com/nazboyko/good-dog/internal/session"
	"github.com/nazboyko/good-dog/internal/soundfx"
)

// Sound serves the dog's own voice by vocalization name, so the client
// asks for /api/sound/playful_bark and never has to know about cache
// keys. Everything else about serving it, including range support, is
// the same path the radio uses.
//
// Silence answers 404 and that is correct: it is a choice with no sound,
// and the client never asks for it.
func Sound(audio http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		effect, ok := soundfx.Find(session.Vocalization(r.PathValue("name")))
		if !ok {
			http.Error(w, "no sound for that", http.StatusNotFound)
			return
		}
		r.SetPathValue("file", soundfx.Key(effect))
		audio(w, r)
	}
}
