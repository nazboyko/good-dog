package httpapi

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
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

func (h *HostVoice) key(text string) string {
	return audiocache.Key(text, h.voiceID, h.settings.String())
}

// Lookup returns the url the client should play, not the disk path.
func (h *HostVoice) Lookup(text string) (string, bool) {
	if h == nil || h.cache == nil {
		return "", false
	}
	key := h.key(text)
	if _, ok := h.cache.Get(key); !ok {
		return "", false
	}
	return "/api/audio/" + key, true
}

// Store synthesizes one line and caches it. Preparation only. The budget
// guard runs before the call, never after, so a runaway loop cannot
// spend the month before anybody notices.
func (h *HostVoice) Store(ctx context.Context, text string) (string, int, error) {
	if h == nil || h.client == nil {
		return "", 0, fmt.Errorf("no voice service configured")
	}
	if err := h.budget.Allow(len(text)); err != nil {
		return "", 0, fmt.Errorf("budget guard: %w", err)
	}
	audio, err := h.client.Synthesize(ctx, text, h.voiceID, h.settings)
	if err != nil {
		return "", 0, err
	}
	key := h.key(text)
	if _, err := h.cache.Put(key, audio); err != nil {
		return "", 0, err
	}
	return "/api/audio/" + filepath.Base(key), len(text), nil
}
