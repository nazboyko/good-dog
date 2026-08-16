package webui

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ground truth is the file in the embedded tree, not a response the
// handler produced, so the test cannot agree with the handler by
// restating its own branching.
func TestIndexIsNeverCached(t *testing.T) {
	h, err := Handler()
	if err != nil {
		t.Fatal(err)
	}
	want, err := fs.ReadFile(built, "dist/index.html")
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{"/", "/anything", "/some/deep/route"}
	served := 0
	for _, path := range paths {
		got := get(t, h, path)
		if got.code != http.StatusOK || got.body != string(want) {
			t.Errorf("%s did not serve the page: %d, %d bytes", path, got.code, len(got.body))
			continue
		}
		served++
		if cc := got.header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("%s served index.html with Cache-Control %q, want no-cache", path, cc)
		}
	}
	// without this the whole test passes by checking nothing the day
	// something stops serving the page at all
	if served != len(paths) {
		t.Fatalf("only %d of %d paths served the page, the rest of this test proved nothing", served, len(paths))
	}
}

// The bundle is safe to cache forever precisely because its name changes
// when its contents do.
func TestHashedAssetsAreCachedHard(t *testing.T) {
	h, err := Handler()
	if err != nil {
		t.Fatal(err)
	}
	assets, err := fs.ReadDir(built, "dist/assets")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) == 0 {
		t.Fatal("the build produced no hashed assets, is the frontend built?")
	}
	for _, a := range assets {
		path := "/assets/" + a.Name()
		got := get(t, h, path)
		if got.code != http.StatusOK {
			t.Errorf("%s returned %d", path, got.code)
			continue
		}
		if cc := got.header.Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
			t.Errorf("%s: Cache-Control %q, want the immutable year", path, cc)
		}
	}
}

// A missing script is a broken build, not a route. Answering it with the
// page turns a 404 into a blank screen with a 200 in the log.
func TestMissingAssetsAre404(t *testing.T) {
	h, err := Handler()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/assets/gone.js", "/missing.css", "/nope.png"} {
		if got := get(t, h, path); got.code != http.StatusNotFound {
			t.Errorf("%s returned %d, want 404", path, got.code)
		}
	}
}

type response struct {
	code   int
	body   string
	header http.Header
}

func get(t *testing.T, h http.Handler, path string) response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	// Result is what a client receives. The recorder's own header map
	// keeps taking writes after the body is gone, so reading that would
	// pass for headers no client ever sees.
	res := rec.Result()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response{code: res.StatusCode, body: string(body), header: res.Header}
}
