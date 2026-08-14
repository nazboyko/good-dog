package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
)

// dev test line, not narrative, players never hear this
const spikeLine = "This is a voice pipeline test for good dog. If you can hear this, the disk cache works."

// SpikeTTS proves one text to speech call landing in the disk cache.
func SpikeTTS(client *elevenlabs.Client, cache *audiocache.Cache, budget *elevenlabs.Budget) http.HandlerFunc {
	settings := elevenlabs.VoiceSettings{Stability: 0.5, SimilarityBoost: 0.75}
	return func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			http.Error(w, "ELEVENLABS_API_KEY not set", http.StatusServiceUnavailable)
			return
		}
		key := audiocache.Key(spikeLine, elevenlabs.DefaultVoiceID, settings.String())
		respond := func(cached bool) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"url": "/api/audio/" + key, "cached": cached})
		}
		if _, ok := cache.Get(key); ok {
			respond(true)
			return
		}
		if err := budget.Allow(len(spikeLine)); err != nil {
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		data, err := client.Synthesize(r.Context(), spikeLine, elevenlabs.DefaultVoiceID, settings)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if _, err := cache.Put(key, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("tts cached %s, %d bytes", key, len(data))
		respond(false)
	}
}
