package radio

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// a cache that records what was asked of it, so a test can prove no
// synthesis happened rather than assuming it
type fakeVoice struct {
	have    map[string]string
	stores  []string
	lookups int
	fail    bool
}

func newFakeVoice() *fakeVoice { return &fakeVoice{have: map[string]string{}} }

func (f *fakeVoice) Lookup(text string) (string, bool) {
	f.lookups++
	file, ok := f.have[text]
	return file, ok
}

func (f *fakeVoice) Store(_ context.Context, text string) (string, int, error) {
	if f.fail {
		return "", 0, errors.New("the voice service said no")
	}
	f.stores = append(f.stores, text)
	file := "cached-" + strings.ToLower(strings.Fields(text)[0]) + ".mp3"
	f.have[text] = file
	return file, len(text), nil
}

func TestPreparingTheHostSetCoversEveryLineOnce(t *testing.T) {
	vc := newFakeVoice()
	got, err := Prepare(context.Background(), vc)
	if err != nil {
		t.Fatal(err)
	}
	if got.Lines != len(HostLines()) || got.Generated != len(HostLines()) {
		t.Errorf("wanted every line generated once: %+v", got)
	}
	if got.Chars == 0 {
		t.Error("a run that generated audio must report what it cost")
	}
	if len(Missing(vc)) != 0 {
		t.Errorf("after preparing, nothing is missing, got %v", Missing(vc))
	}
}

// A warm cache is the normal case on every boot after the first. It must
// cost nothing and generate nothing.
func TestPreparingTwiceCostsNothingTheSecondTime(t *testing.T) {
	vc := newFakeVoice()
	if _, err := Prepare(context.Background(), vc); err != nil {
		t.Fatal(err)
	}
	before := len(vc.stores)
	second, err := Prepare(context.Background(), vc)
	if err != nil {
		t.Fatal(err)
	}
	if len(vc.stores) != before {
		t.Errorf("a warm cache generated %d more lines", len(vc.stores)-before)
	}
	if second.Generated != 0 || second.Chars != 0 {
		t.Errorf("a warm run must cost nothing: %+v", second)
	}
	if second.AlreadyIn != len(HostLines()) {
		t.Errorf("every line should have been found: %+v", second)
	}
}

// The rule this ship exists to hold: playing a night never synthesizes.
// WithVoices only reads, so a missing recording comes back as a named
// miss and never as a quiet paid network call under a live listener.
func TestPlayingANightNeverSynthesizes(t *testing.T) {
	vc := newFakeVoice()
	cues := Broadcast(testPool(), []string{"Her name is Venus."})

	// nothing prepared: every host line is reported missing, none stored
	voiced, missing := WithVoices(cues, vc)
	if len(vc.stores) != 0 {
		t.Errorf("play time synthesized %d lines, it must never synthesize", len(vc.stores))
	}
	if len(missing) != len(HostLines()) {
		t.Errorf("an empty cache should report every host line missing, got %v", missing)
	}
	for _, c := range voiced {
		if c.Audio != "" {
			t.Errorf("a line with no recording must not claim one: %q", c.Line)
		}
		if strings.TrimSpace(c.Line) == "" {
			t.Error("a missing recording must never cost the line itself")
		}
	}

	// prepared: the host speaks, the dog stories stay text
	if _, err := Prepare(context.Background(), vc); err != nil {
		t.Fatal(err)
	}
	stores := len(vc.stores)
	voiced, missing = WithVoices(cues, vc)
	if len(missing) != 0 {
		t.Errorf("after preparing nothing should be missing, got %v", missing)
	}
	if len(vc.stores) != stores {
		t.Error("attaching voices generated audio, it must only read")
	}
	var spoken, silent int
	for _, c := range voiced {
		if c.Audio != "" {
			spoken++
			if c.Speaker != SpeakerRanger {
				t.Errorf("only the host is voiced in this ship, got %s: %q", c.Speaker, c.Line)
			}
		} else {
			silent++
		}
	}
	if spoken != len(HostLines()) {
		t.Errorf("every host line should be voiced, got %d", spoken)
	}
	if silent == 0 {
		t.Error("the dog stories are text in this ship, something voiced them")
	}
}

// Every line stays on screen whether or not it has a recording, because
// the night has to be legible with the sound off.
func TestEveryCueKeepsItsTextWithOrWithoutAudio(t *testing.T) {
	vc := newFakeVoice()
	if _, err := Prepare(context.Background(), vc); err != nil {
		t.Fatal(err)
	}
	cues := Broadcast(testPool(), []string{"Her name is Venus."})
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

// A voice service having a bad night is not the game having a bad night.
func TestAFailedPreparationIsReportedNotSwallowed(t *testing.T) {
	vc := newFakeVoice()
	vc.fail = true
	got, err := Prepare(context.Background(), vc)
	if err == nil {
		t.Fatal("a failure to prepare must be reported to the caller")
	}
	if got.Generated != 0 {
		t.Errorf("nothing was generated, the report should say so: %+v", got)
	}
	if len(Missing(vc)) != len(HostLines()) {
		t.Error("every line is still missing after a failed run")
	}
	// and the night still builds, silent
	voiced, missing := WithVoices(Broadcast(testPool(), nil), vc)
	if len(voiced) == 0 {
		t.Fatal("a silent night is still a night")
	}
	if len(missing) == 0 {
		t.Error("the misses must be named so this shows up as a bug")
	}
}

func TestNoVoiceCacheAtAllLeavesTheNightIntact(t *testing.T) {
	cues := Broadcast(testPool(), nil)
	voiced, missing := WithVoices(cues, nil)
	if len(voiced) != len(cues) || len(missing) != 0 {
		t.Errorf("with no cache the night passes through untouched: %d cues, %v", len(voiced), missing)
	}
}
