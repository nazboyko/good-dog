package httpapi

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nazboyko/good-dog/internal/radio"
)

type fixedNightly []radio.Cue

func (f fixedNightly) Tonight(context.Context, string) []radio.Cue { return f }

// readEvents pulls SSE frames off the wire until the reader is done.
// Comments (the pings) are skipped, everything else comes back as
// "event: data" pairs in arrival order.
func readEvents(t *testing.T, body *bufio.Reader, want int) []string {
	t.Helper()
	var out []string
	var event string
	for len(out) < want {
		line, err := body.ReadString('\n')
		if err != nil {
			return out
		}
		line = strings.TrimRight(line, "\n")
		switch {
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			out = append(out, event+": "+strings.TrimPrefix(line, "data: "))
		}
	}
	return out
}

// The stream is the radio. Cues arrive in order, paced by the server,
// each one flushed on its own so a client sees it the moment it lands.
func TestTheStreamPacesTheNight(t *testing.T) {
	cues := []radio.Cue{
		{At: 10 * time.Millisecond, Speaker: radio.SpeakerRanger, Line: "Shelter radio, and it is late."},
		{At: 40 * time.Millisecond, Speaker: radio.SpeakerStory, Line: "That is Keno, over at Animal Humane Society in Golden Valley."},
		{At: 70 * time.Millisecond, Speaker: radio.SpeakerRanger, Line: "Sleep if you can."},
	}
	srv := httptest.NewServer(Events(fixedNightly(cues)))
	defer srv.Close()

	start := time.Now()
	res, err := http.Get(srv.URL + "/events?session=abc")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("content type %q", ct)
	}
	// gzip would hold the whole stream in a buffer and nothing would arrive
	if enc := res.Header.Get("Content-Encoding"); enc != "" {
		t.Errorf("the stream must never be compressed, got %q", enc)
	}

	// hello, three cues, then the closing marker
	got := readEvents(t, bufio.NewReader(res.Body), 5)
	if len(got) != 5 {
		t.Fatalf("wanted hello plus three cues plus done, got %d: %v", len(got), got)
	}
	if !strings.HasPrefix(got[0], "hello: ") {
		t.Errorf("every connection opens with hello, got %q", got[0])
	}
	for i, want := range []string{"Shelter radio", "That is Keno", "Sleep if you can"} {
		if !strings.HasPrefix(got[i+1], "radio: ") || !strings.Contains(got[i+1], want) {
			t.Errorf("cue %d should carry %q, got %q", i, want, got[i+1])
		}
		if !strings.Contains(got[i+1], `"index":`) || !strings.Contains(got[i+1], `"total":3`) {
			t.Errorf("cue %d must say where it sits in the night: %q", i, got[i+1])
		}
	}
	if !strings.HasPrefix(got[4], "radio_done: ") {
		t.Errorf("the night ends with a marker, got %q", got[4])
	}
	// paced, not dumped: the last cue cannot have arrived before its offset
	if elapsed := time.Since(start); elapsed < 70*time.Millisecond {
		t.Errorf("the whole night arrived in %v, the server is not pacing it", elapsed)
	}
}

// A session with no night, or no session at all, is a connection that
// works and simply has nothing to say. Never an error screen.
func TestTheStreamOpensEvenWithNothingOn(t *testing.T) {
	srv := httptest.NewServer(Events(fixedNightly(nil)))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	got := readEvents(t, bufio.NewReader(res.Body), 1)
	if len(got) != 1 || !strings.HasPrefix(got[0], "hello: ") {
		t.Errorf("a quiet night still says hello, got %v", got)
	}
}

// A player who closes the tab mid broadcast must not leave the handler
// sleeping through the rest of the night.
func TestTheStreamStopsWhenTheListenerLeaves(t *testing.T) {
	cues := []radio.Cue{
		{At: 10 * time.Millisecond, Speaker: radio.SpeakerRanger, Line: "first"},
		{At: 30 * time.Second, Speaker: radio.SpeakerRanger, Line: "much later"},
	}
	rec := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/events?session=abc", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		Events(fixedNightly(cues))(rec, req)
		close(done)
	}()
	time.Sleep(40 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler is still waiting on a cue nobody is listening for")
	}
	if !strings.Contains(rec.Body.String(), "first") {
		t.Error("the cue that was due should still have been written")
	}
	if strings.Contains(rec.Body.String(), "much later") {
		t.Error("a cue after the listener left must not be written")
	}
}
