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

func (f *fakeVoice) Lookup(text string, _ Voice) (string, bool) {
	f.lookups++
	file, ok := f.have[text]
	return file, ok
}

func (f *fakeVoice) Store(_ context.Context, text string, _ Voice) (string, int, error) {
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
	cues := Broadcast(testPool(), []string{"Her name is Venus."}, Ranger)

	// nothing prepared: every host line is reported missing, none stored
	voiced, missing := WithVoices(cues, vc)
	if len(vc.stores) != 0 {
		t.Errorf("play time synthesized %d lines, it must never synthesize", len(vc.stores))
	}
	// every line in the night, host and dog alike, is a miss on an
	// empty cache: the whole broadcast is recorded ahead of time
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

	// prepared: the host speaks, the dog stories stay text
	if _, err := Prepare(context.Background(), vc); err != nil {
		t.Fatal(err)
	}
	// Prepare covers the host's fixed lines only. Everything a dog says,
	// and every line naming a dog, is per dog and belongs to the
	// pregeneration command, so it is still a named miss here.
	stores := len(vc.stores)
	voiced, missing = WithVoices(cues, vc)
	if len(vc.stores) != stores {
		t.Error("attaching voices generated audio, it must only read")
	}
	var spoken int
	for _, c := range voiced {
		if c.Audio == "" {
			continue
		}
		spoken++
		if c.Line != HostOpen && c.Line != HostWhoIsUp && c.Line != HostClose {
			t.Errorf("Prepare only covers the fixed host lines, but %q is voiced", c.Line)
		}
	}
	if spoken != len(HostLines()) {
		t.Errorf("the three fixed host lines should be voiced, got %d", spoken)
	}
	if len(missing) != len(cues)-len(HostLines()) {
		t.Errorf("everything per dog is still missing, got %d of %d", len(missing), len(cues)-len(HostLines()))
	}
}

// Every line stays on screen whether or not it has a recording, because
// the night has to be legible with the sound off.
func TestEveryCueKeepsItsTextWithOrWithoutAudio(t *testing.T) {
	vc := newFakeVoice()
	if _, err := Prepare(context.Background(), vc); err != nil {
		t.Fatal(err)
	}
	cues := Broadcast(testPool(), []string{"Her name is Venus."}, Ranger)
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
	voiced, missing := WithVoices(Broadcast(testPool(), nil, Ranger), vc)
	if len(voiced) == 0 {
		t.Fatal("a silent night is still a night")
	}
	if len(missing) == 0 {
		t.Error("the misses must be named so this shows up as a bug")
	}
}

func TestNoVoiceCacheAtAllLeavesTheNightIntact(t *testing.T) {
	cues := Broadcast(testPool(), nil, Ranger)
	voiced, missing := WithVoices(cues, nil)
	if len(voiced) != len(cues) || len(missing) != 0 {
		t.Errorf("with no cache the night passes through untouched: %d cues, %v", len(voiced), missing)
	}
}
