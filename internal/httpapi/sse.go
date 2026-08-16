package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nazboyko/good-dog/internal/radio"
)

// Events serves the single multiplexed SSE stream. Every server pushed
// event goes through here with an event type, never one stream per feature.
//
// The night radio is paced here rather than in the browser, so the
// broadcast keeps its own time whatever the tab is doing. The client
// only plays what arrives. It also holds the cue list from the view, so
// a client that never manages to connect can still run the night off
// its own timer, which is a worse night but not a blank screen.
func Events(nightly Nightly) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		// sent on every connect, so EventSource reconnects heal themselves
		fmt.Fprintf(w, "event: hello\ndata: {\"server_time\":%q}\n\n", time.Now().UTC().Format(time.RFC3339))
		flusher.Flush()

		cues := nightly.Tonight(r.Context(), r.URL.Query().Get("session"))
		stream(r.Context(), w, flusher, cues)
	}
}

// Nightly is how the stream finds out what is on tonight. Kept as an
// interface so the handler can be tested without a store or a provider.
type Nightly interface {
	Tonight(ctx context.Context, sessionID string) []radio.Cue
}

// stream writes each cue when it comes due and then holds the
// connection open with pings. Offsets are measured from the moment the
// client connected, so the broadcast starts when somebody is listening.
func stream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, cues []radio.Cue) {
	start := time.Now()
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	for i, cue := range cues {
		wait := cue.At - time.Since(start)
		if wait > 0 {
			timer.Reset(wait)
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
		payload, err := json.Marshal(struct {
			Index int       `json:"index"`
			Total int       `json:"total"`
			Cue   radio.Cue `json:"cue"`
		}{i, len(cues), cue})
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "event: radio\nid: %d\ndata: %s\n\n", i, payload)
		flusher.Flush()
	}
	if len(cues) > 0 {
		fmt.Fprint(w, "event: radio_done\ndata: {}\n\n")
		flusher.Flush()
	}

	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
