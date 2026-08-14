package audiocache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Disk cache for generated speech. Same text, voice and settings never
// cost credits twice, see docs/tech-stack.md.

type Cache struct {
	root string
}

func New(root string) *Cache {
	return &Cache{root: root}
}

// Key derives the cache file name from the inputs that vary per call,
// the model id must join the key if it ever becomes a parameter.
func Key(text, voiceID, settings string) string {
	sum := sha256.Sum256([]byte(text + "\x00" + voiceID + "\x00" + settings))
	return hex.EncodeToString(sum[:8]) + ".mp3"
}

// Get reports whether the key is already on disk.
func (c *Cache) Get(key string) (string, bool) {
	path := filepath.Join(c.root, key)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// Put writes audio bytes under the key, creating the cache dir on first use.
func (c *Cache) Put(key string, data []byte) (string, error) {
	if err := os.MkdirAll(c.root, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}
	path := filepath.Join(c.root, key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write cache file: %w", err)
	}
	return path, nil
}
