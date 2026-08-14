package httpapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSafeAudioName(t *testing.T) {
	good := []string{"test-tone.mp3", "abc_123.mp3", "a.b.mp3"}
	bad := []string{"", "tone.wav", "../secret.mp3", "a/b.mp3", "UPPER.mp3", "sp ace.mp3", "..mp3"}
	for _, name := range good {
		if !safeAudioName(name) {
			t.Errorf("safeAudioName(%q) = false, want true", name)
		}
	}
	for _, name := range bad {
		if safeAudioName(name) {
			t.Errorf("safeAudioName(%q) = true, want false", name)
		}
	}
}

func TestAudioServesRanges(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "t.mp3"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := Audio(dir)

	req := httptest.NewRequest("GET", "/api/audio/t.mp3", nil)
	req.SetPathValue("file", "t.mp3")
	req.Header.Set("Range", "bytes=2-5")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 206 {
		t.Fatalf("status = %d, want 206", rec.Code)
	}
	if got := rec.Body.String(); got != "2345" {
		t.Errorf("body = %q, want %q", got, "2345")
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Errorf("Content-Range = %q, want %q", got, "bytes 2-5/10")
	}
}
