package httpapi

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
	"github.com/nazboyko/good-dog/internal/radio"
)

// HostVoice is Ranger, backed by the disk cache and, only during
// preparation, by the speech service. It is the radio.VoiceCache the
// night reads from.
//
// Lookup never touches the network and never costs a credit, which is
// what makes it safe to call while a broadcast is playing. Store is the
// only path that spends anything and is only reached from preparation
// at boot.
type HostVoice struct {
	cache    *audiocache.Cache
	client   *elevenlabs.Client
	budget   *elevenlabs.Budget
	voiceID  string
	settings elevenlabs.VoiceSettings
}

func NewHostVoice(cache *audiocache.Cache, client *elevenlabs.Client, budget *elevenlabs.Budget, voiceID string, settings elevenlabs.VoiceSettings) *HostVoice {
	return &HostVoice{cache: cache, client: client, budget: budget, voiceID: voiceID, settings: settings}
}

// KeyFor is the cache name for one line in one voice. The voice and its
// settings are part of the key because the same sentence read by the
// host and by a dog, or by two dogs sharing a library voice at different
// stabilities, are different recordings. Exported so the recorder plans
// against the same key the night looks up.
func KeyFor(text string, v radio.Voice) string {
	settings := elevenlabs.VoiceSettings{Stability: v.Stability, SimilarityBoost: v.Similarity}
	return audiocache.Key(text, v.ID, settings.String())
}

// Lookup returns the url the client should play, not the disk path.
func (h *HostVoice) Lookup(text string, v radio.Voice) (string, bool) {
	if h == nil || h.cache == nil {
		return "", false
	}
	key := KeyFor(text, v)
	if _, ok := h.cache.Get(key); !ok {
		return "", false
	}
	return "/api/audio/" + key, true
}

// Store synthesizes one line and caches it. Preparation only. The budget
// guard runs before the call, never after, so a runaway loop cannot
// spend the month before anybody notices.
func (h *HostVoice) Store(ctx context.Context, text string, v radio.Voice) (string, int, error) {
	if h == nil || h.client == nil {
		return "", 0, fmt.Errorf("no voice service configured")
	}
	if err := h.budget.Allow(len(text)); err != nil {
		return "", 0, fmt.Errorf("budget guard: %w", err)
	}
	settings := elevenlabs.VoiceSettings{Stability: v.Stability, SimilarityBoost: v.Similarity}
	audio, err := h.client.Synthesize(ctx, text, v.ID, settings)
	if err != nil {
		return "", 0, err
	}
	key := KeyFor(text, v)
	if _, err := h.cache.Put(key, audio); err != nil {
		return "", 0, err
	}
	return "/api/audio/" + filepath.Base(key), len(text), nil
}
