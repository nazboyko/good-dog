package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Audio serves mp3 files by name from the first root that has them.
// http.ServeContent gives the range support browsers need to seek and
// to start playback fast.
func Audio(roots ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("file")
		if !safeAudioName(name) {
			http.Error(w, "bad audio name", http.StatusBadRequest)
			return
		}
		var f *os.File
		var err error
		for _, root := range roots {
			if f, err = os.Open(filepath.Join(root, name)); err == nil {
				break
			}
		}
		if err != nil {
			http.Error(w, "audio not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			http.Error(w, "audio not readable", http.StatusInternalServerError)
			return
		}
		// explicit type because the distroless prod image has no mime table
		w.Header().Set("Content-Type", "audio/mpeg")
		http.ServeContent(w, r, name, info.ModTime(), f)
	}
}

// safeAudioName allows plain mp3 file names only, no paths, no traversal.
func safeAudioName(name string) bool {
	if name == "" || !strings.HasSuffix(name, ".mp3") {
		return false
	}
	for _, r := range name {
		ok := r == '.' || r == '-' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return !strings.Contains(name, "..")
}
