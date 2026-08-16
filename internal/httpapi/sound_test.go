package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nazboyko/good-dog/internal/soundfx"
)

// The route is what the browser actually asks for, so the test asks for
// it the same way: by vocalization name, never by cache key.
func TestTheDogSoundsWhenAskedByName(t *testing.T) {
	dir := t.TempDir()
	for _, e := range soundfx.All() {
		body := []byte("mp3 for " + string(e.Vocalization))
		if err := os.WriteFile(filepath.Join(dir, soundfx.Key(e)), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/sound/{name}", Sound(Audio(dir)))

	for _, e := range soundfx.All() {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/sound/"+string(e.Vocalization), nil))
		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Errorf("%s: %d, a pressed button made no sound", e.Vocalization, res.StatusCode)
			continue
		}
		if got := rec.Body.String(); got != "mp3 for "+string(e.Vocalization) {
			t.Errorf("%s served the wrong recording: %q", e.Vocalization, got)
		}
		if ct := res.Header.Get("Content-Type"); ct != "audio/mpeg" {
			t.Errorf("%s served as %q", e.Vocalization, ct)
		}
		if res.Header.Get("Accept-Ranges") != "bytes" {
			t.Errorf("%s served without range support", e.Vocalization)
		}
	}
}

// The whitelist has to be what refuses these, not a missing file.
//
// A handler that stopped checking would fall through with the zero
// Effect and ask the file layer for its key, so that file is planted
// here on purpose. Now the only thing standing between an unknown name
// and a 200 is the check itself, and a 404 can only have come from it.
func TestSilenceHasNoSoundToServe(t *testing.T) {
	dir := t.TempDir()
	for _, e := range soundfx.All() {
		if err := os.WriteFile(filepath.Join(dir, soundfx.Key(e)), []byte("mp3"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, soundfx.Key(soundfx.Effect{})), []byte("not a dog"), 0o644); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/sound/{name}", Sound(Audio(dir)))
	for _, name := range []string{"silence", "nonsense", "", "PLAYFUL_BARK"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/sound/"+name, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%q returned %d, want 404", name, rec.Code)
		}
	}
}
