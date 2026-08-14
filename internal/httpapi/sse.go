package httpapi

import (
	"fmt"
	"net/http"
	"time"
)

// Events serves the single multiplexed SSE stream. Every server pushed
// event goes through here with an event type, never one stream per feature.
func Events() http.HandlerFunc {
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

		ping := time.NewTicker(20 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ping.C:
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	}
}
