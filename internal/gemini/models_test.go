package gemini

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Error("api key header missing")
		}
		fmt.Fprint(w, `{"models":[{"name":"models/gemini-3.6-flash"},{"name":"models/gemini-pro-x"}]}`)
	}))
	defer srv.Close()

	got, err := newTestClient(srv.URL).listModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "models/gemini-3.6-flash" {
		t.Errorf("got %v", got)
	}
}

func TestModelPinWarning(t *testing.T) {
	models := []string{
		"models/gemini-3.5-flash",
		"models/gemini-3.6-flash",
		"models/gemini-pro-x",
		"models/embedding-001",
	}
	if warning, missing := modelPinWarning(models, "gemini-3.6-flash"); missing {
		t.Errorf("pin present must not warn, got %q", warning)
	}
	warning, missing := modelPinWarning(models, "gemini-9.9-flash")
	if !missing {
		t.Fatal("absent pin must warn")
	}
	for _, want := range []string{"gemini-9.9-flash", "gemini-3.5-flash", "gemini-3.6-flash"} {
		if !strings.Contains(warning, want) {
			t.Errorf("warning must name %s, got %q", want, warning)
		}
	}
	if strings.Contains(warning, "gemini-pro-x") || strings.Contains(warning, "embedding-001") {
		t.Errorf("warning must list flash models only, got %q", warning)
	}
}
