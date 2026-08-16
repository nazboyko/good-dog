package soundfx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nazboyko/good-dog/internal/session"
)

// Ground truth is the game's own list, read from session rather than
// copied here, so a vocalization added to the game with no sound and no
// decision about it shows up as a failure instead of a dead button.
func TestEveryVocalizationIsAccountedFor(t *testing.T) {
	everyVocalization := session.Vocalizations()
	for _, v := range everyVocalization {
		_, ok := Find(v)
		if v == session.Silence {
			if ok {
				t.Error("silence has a sound, it is meant to be the quiet one")
			}
			continue
		}
		if !ok {
			t.Errorf("%s has no sound, so pressing it happens in silence", v)
		}
	}
	if len(All()) != len(everyVocalization)-1 {
		t.Errorf("%d sounds for %d vocalizations, one of which is silence", len(All()), len(everyVocalization))
	}
}

// Two sounds sharing a key means one dog noise plays for two different
// presses, which is worse than no sound at all.
func TestEverySoundHasItsOwnKey(t *testing.T) {
	seen := map[string]session.Vocalization{}
	for _, e := range All() {
		key := Key(e)
		if other, clash := seen[key]; clash {
			t.Errorf("%s and %s share the cache key %s", e.Vocalization, other, key)
		}
		seen[key] = e.Vocalization
	}
}

// A local guard, not a CI one: cache/ is gitignored so a fresh checkout
// skips this. It still gates the deploy, because the image is built from
// this working tree. What runs where players are is soundfx.Missing at
// boot. If this fails, either make voices has not been run or an input
// changed without a re-record, and a press would make no sound.
func TestEverySoundIsRecorded(t *testing.T) {
	root := filepath.Join("..", "..", "cache", "audio")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no audio cache here, nothing to check against")
	}
	for _, e := range All() {
		path := filepath.Join(root, Key(e))
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("%s has no recording at %s, run make voices", e.Vocalization, path)
			continue
		}
		// an empty or near empty mp3 is a failed call that got cached
		if info.Size() < 1024 {
			t.Errorf("%s is only %d bytes, that is not a dog", e.Vocalization, info.Size())
		}
	}
}

// Anything the model is given decides what comes back, so anything the
// model is given has to move the key. A change that does not move it
// leaves make voices reporting a warm cache and the game playing the old
// sound for good.
func TestEveryGenerationInputMovesTheKey(t *testing.T) {
	e, ok := Find(session.Howl)
	if !ok {
		t.Fatal("setup: no howl")
	}
	reworded := e
	reworded.Prompt += " and again"
	if Key(reworded) == Key(e) {
		t.Error("a different prompt kept the same recording")
	}
	longer := e
	longer.Seconds += 2
	if Key(longer) == Key(e) {
		t.Error("a different duration kept the same recording")
	}
}
