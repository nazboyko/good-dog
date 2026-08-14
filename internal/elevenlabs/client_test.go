package elevenlabs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSynthesize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "test-key" {
			t.Error("api key header missing")
		}
		if !strings.Contains(r.URL.Path, DefaultVoiceID) {
			t.Errorf("voice missing from path %s", r.URL.Path)
		}
		if r.URL.Query().Get("output_format") != "mp3_44100_128" {
			t.Error("output format missing")
		}
		w.Write([]byte("fake mp3"))
	}))
	defer srv.Close()

	c := New("test-key")
	c.baseURL = srv.URL
	data, err := c.Synthesize(context.Background(), "hello", DefaultVoiceID, VoiceSettings{Stability: 0.5, SimilarityBoost: 0.75})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake mp3" {
		t.Errorf("got %q", data)
	}
}

func TestBudgetAllow(t *testing.T) {
	var b Budget
	if err := b.Allow(0); err == nil {
		t.Error("zero chars should fail")
	}
	if err := b.Allow(MaxCallChars + 1); err == nil {
		t.Error("over the per call limit should fail")
	}
	if err := b.Allow(100); err != nil {
		t.Errorf("normal call failed: %v", err)
	}
	for range 24 {
		b.Allow(2000)
	}
	if err := b.Allow(2000); err == nil {
		t.Error("over the total limit should fail")
	}
	if err := b.Allow(100); err != nil {
		t.Errorf("failed reservation must not eat budget: %v", err)
	}
}
