package dogsheet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nazboyko/good-dog/internal/animal"
)

// Dog sheets are cached forever. The key carries the description hash
// so an updated listing compiles fresh instead of serving stale truth.
type Cache struct {
	root string
}

func NewCache(root string) *Cache {
	return &Cache{root: root}
}

func (c *Cache) key(a animal.Animal) string {
	sum := sha256.Sum256([]byte(a.Description))
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(a.ID))
	return safe + "-" + hex.EncodeToString(sum[:4]) + ".json"
}

func (c *Cache) Get(a animal.Animal) (*DogSheet, bool) {
	raw, err := os.ReadFile(filepath.Join(c.root, c.key(a)))
	if err != nil {
		return nil, false
	}
	var sheet DogSheet
	if err := json.Unmarshal(raw, &sheet); err != nil {
		return nil, false
	}
	// id sanitization could collide, never serve another dog's sheet
	if sheet.AnimalID != a.ID {
		return nil, false
	}
	return &sheet, true
}

func (c *Cache) Put(a animal.Animal, sheet *DogSheet) error {
	if err := os.MkdirAll(c.root, 0o755); err != nil {
		return fmt.Errorf("create sheet cache dir: %w", err)
	}
	raw, err := json.MarshalIndent(sheet, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.root, c.key(a)), raw, 0o644)
}
