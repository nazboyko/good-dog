package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/nazboyko/good-dog/internal/gemini"
)

// fictional note, written for the spike, no real dog involved
const spikeNote = `Biscuit is a three year old lab mix. Loves tennis balls and belly rubs. A bit shy the first day, warms up fast. Walks nicely on leash, knows sit.`

// SpikeGemini proves one grounded extraction call end to end.
func SpikeGemini(client *gemini.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			http.Error(w, "GEMINI_API_KEY not set", http.StatusServiceUnavailable)
			return
		}
		out, err := client.ExtractNote(r.Context(), spikeNote)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"note": spikeNote, "extraction": out})
	}
}
