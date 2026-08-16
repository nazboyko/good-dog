package radio

import (
	"context"
	"strings"
	"testing"
)

// a cache that records what was asked of it, so a test can prove no
// synthesis happened rather than assuming it
type fakeVoice struct {
	have   map[string]string
	stores []string
}

func newFakeVoice() *fakeVoice { return &fakeVoice{have: map[string]string{}} }

func cacheKey(text string, v Voice) string { return v.ID + "\x00" + text }

func (f *fakeVoice) Lookup(text string, v Voice) (string, bool) {
	file, ok := f.have[cacheKey(text, v)]
	return file, ok
}

// Store stands in for the recording command. Nothing the game does at
// play time calls it: if a test sees a store during a night, that is
// the bug this whole design exists to prevent.
func (f *fakeVoice) Store(_ context.Context, text string, v Voice) (string, int, error) {
	f.stores = append(f.stores, text)
	f.have[cacheKey(text, v)] = "cached.mp3"
	return "cached.mp3", len(text), nil
}

// record is what `make voices` does: every line the night can hold, in
// the voice it is written for.
func (f *fakeVoice) record(cues []Cue) {
	for _, c := range cues {
		f.have[cacheKey(c.Line, c.Voice)] = "cached.mp3"
	}
}

// Playing a night never synthesizes. WithVoices only reads, so a
// missing recording comes back as a named miss and never as a quiet
// paid network call under a live listener.
func TestPlayingANightNeverSynthesizes(t *testing.T) {
	vc := newFakeVoice()
	own := []string{"This one is for the dog in the third kennel down.", "Her name is Venus."}
	cues := Broadcast(testPool(), own, Ranger, 0)

	// nothing recorded: every line is a named miss and nothing is stored
	voiced, missing := WithVoices(cues, vc)
	if len(vc.stores) != 0 {
		t.Errorf("play time synthesized %d lines, it must never synthesize", len(vc.stores))
	}
	if len(missing) != len(cues) {
		t.Errorf("an empty cache should report all %d lines missing, got %d", len(cues), len(missing))
	}
	for _, c := range voiced {
		if c.Audio != "" {
			t.Errorf("a line with no recording must not claim one: %q", c.Line)
		}
		if strings.TrimSpace(c.Line) == "" {
			t.Error("a missing recording must never cost the line itself")
		}
	}

	// recorded: every line plays, and still nothing was synthesized
	vc.record(cues)
	voiced, missing = WithVoices(cues, vc)
	if len(vc.stores) != 0 {
		t.Error("attaching voices generated audio, it must only read")
	}
	if len(missing) != 0 {
		t.Errorf("after recording nothing should be missing, got %v", missing)
	}
	for _, c := range voiced {
		if c.Audio == "" {
			t.Errorf("a recorded line has no audio attached: %q", c.Line)
		}
	}
}

// The same sentence read by the host and by a dog are two different
// recordings, so the cache key carries the voice. Keyed on text alone a
// dog would play the host's read of its own line.
func TestTheCacheKeyCarriesTheVoice(t *testing.T) {
	vc := newFakeVoice()
	line := "Loves to snuggle up close with her people."
	dogVoice := VoiceFor(dog("6 lbs", "Senior"), nil)
	if dogVoice.ID == Ranger.ID {
		t.Fatal("setup: the dog and the host must be different voices")
	}

	vc.have[cacheKey(line, Ranger)] = "host.mp3"
	if _, ok := vc.Lookup(line, dogVoice); ok {
		t.Error("a dog picked up the host's recording of its own line")
	}
	if _, ok := vc.Lookup(line, Ranger); !ok {
		t.Error("the host's own recording went missing")
	}
}

// Every line stays on screen whether or not it has a recording, because
// the night has to be legible with the sound off.
func TestEveryCueKeepsItsTextWithOrWithoutAudio(t *testing.T) {
	vc := newFakeVoice()
	cues := Broadcast(testPool(), []string{"Her name is Venus."}, Ranger, 0)
	vc.record(cues[:3]) // only part of it recorded
	voiced, _ := WithVoices(cues, vc)
	if len(voiced) != len(cues) {
		t.Fatalf("attaching audio changed the night: %d cues became %d", len(cues), len(voiced))
	}
	for i, c := range voiced {
		if c.Line != cues[i].Line || c.At != cues[i].At || c.Speaker != cues[i].Speaker {
			t.Errorf("cue %d changed when its audio was attached: %+v vs %+v", i, cues[i], c)
		}
	}
}

// A silent night is a worse night, and a much better one than no night.
func TestAnUnrecordedNightStillPlays(t *testing.T) {
	vc := newFakeVoice()
	voiced, missing := WithVoices(Broadcast(testPool(), nil, Ranger, 0), vc)
	if len(voiced) == 0 {
		t.Fatal("a silent night is still a night")
	}
	if len(missing) == 0 {
		t.Error("the misses must be named so this shows up as a bug")
	}
	// and the boot check sees the whole rotation, not just tonight's draw
	if len(Missing(vc)) != len(HostLines()) {
		t.Errorf("every host line is unrecorded, got %d of %d", len(Missing(vc)), len(HostLines()))
	}
}

func TestNoVoiceCacheAtAllLeavesTheNightIntact(t *testing.T) {
	cues := Broadcast(testPool(), nil, Ranger, 0)
	voiced, missing := WithVoices(cues, nil)
	if len(voiced) != len(cues) || len(missing) != 0 {
		t.Errorf("with no cache the night passes through untouched: %d cues, %v", len(voiced), missing)
	}
}
