// Package webui serves the built frontend out of the binary.
//
// One artifact ships: the game, the API, the stream and the audio all
// come from the same process, which is what makes a deploy a single
// file on a single machine and a rollback a single previous file.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// The build output. Committed as a build step, not to git: the
// Dockerfile runs the vite build before the go build, and
// `make embed-check` proves the directory is populated before a deploy
// can silently ship an empty page.
//
//go:embed all:dist
var built embed.FS

// Handler serves the built app. Unknown paths fall back to index.html
// because the client owns its own routing, but only for requests that
// look like navigation: a missing asset must 404 rather than quietly
// return HTML, which is how a broken bundle turns into a blank screen
// with a 200 next to it in the log.
func Handler() (http.Handler, error) {
	dist, err := fs.Sub(built, "dist")
	if err != nil {
		return nil, err
	}
	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(dist, clean); err == nil {
			// the bundle is content hashed, so it can be cached hard.
			// index.html cannot: it is what points at the new hashes.
			if strings.HasPrefix(clean, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}
		if isAsset(clean) {
			http.NotFound(w, r)
			return
		}
		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "the game is not built into this binary", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
	}), nil
}

// isAsset is anything with a file extension: a request for a script or
// an image that is not there is a mistake, not a route.
func isAsset(path string) bool {
	last := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		last = path[i+1:]
	}
	return strings.Contains(last, ".")
}
