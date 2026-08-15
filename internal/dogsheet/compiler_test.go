package dogsheet

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/gemini"
)

var testDog = animal.Animal{
	ID: "test-1", Name: "Biscuit", Breed: "Lab mix", AgeGroup: "Adult",
	Sex: "Female", Size: "Large",
	Description: "Loves tennis balls and belly rubs. A bit shy the first day. Scared of thunder, hides under the bed. Walks nicely on leash.",
	RetrievedAt: time.Date(2026, 8, 14, 23, 0, 0, 0, time.UTC),
}

// fakeLLM plays back scripted responses in order.
type fakeLLM struct {
	responses []string
	errs      []error
	calls     int
	prompts   []string
}

func (f *fakeLLM) GenerateJSON(ctx context.Context, prompt string, opts gemini.Options) ([]byte, error) {
	i := f.calls
	f.calls++
	f.prompts = append(f.prompts, prompt)
	if i < len(f.errs) && f.errs[i] != nil {
		return nil, f.errs[i]
	}
	if i < len(f.responses) {
		return []byte(f.responses[i]), nil
	}
	return nil, &gemini.APIError{Status: 500, Detail: "script exhausted"}
}

const goodExtraction = `{"facts":["Loves tennis balls and belly rubs","A bit shy the first day","Scared of thunder, hides under the bed"]}`

const goodSheet = `{
  "matrix": {
    "energy": {"level": "high", "cites": ["f6"]},
    "confidence": {"level": "low", "cites": ["f7"]},
    "sociability": {"level": "medium", "cites": []},
    "patience": {"level": "medium", "cites": []},
    "food_drive": {"level": "medium", "cites": []},
    "noise_sensitivity": {"level": "high", "cites": ["f8"]}
  },
  "fears": [{"value": "loud rumbles from the sky", "quote": "Scared of thunder, hides under the bed", "cites": ["f8"]}],
  "quirks": [{"value": "brings the ball back twice", "cites": ["f6"]}],
  "movement": {"value": "keeps low the first day, then bounces", "cites": ["f7"]},
  "voice": {"value": "soft huffs, big tail", "cites": ["f6"]},
  "radio_seed": {"value": "Somewhere in the big room, Biscuit is dreaming about tennis balls.", "cites": ["f1", "f6"]}
}`

func newTestCompiler(t *testing.T, llm generator) *Compiler {
	t.Helper()
	return NewCompiler(llm, NewCache(t.TempDir()))
}

func TestCompileHappyPath(t *testing.T) {
	llm := &fakeLLM{responses: []string{goodExtraction, goodSheet}}
	sheet, err := newTestCompiler(t, llm).Compile(context.Background(), testDog)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.Default {
		t.Fatal("happy path must not serve the default sheet")
	}
	if len(sheet.Facts) != 8 {
		t.Errorf("facts = %d, want 5 structured plus 3 quoted", len(sheet.Facts))
	}
	if len(sheet.Fears) != 1 || sheet.Fears[0].Quote != "Scared of thunder, hides under the bed" {
		t.Errorf("fear lost: %+v", sheet.Fears)
	}
	if llm.calls != 2 {
		t.Errorf("calls = %d, want 2", llm.calls)
	}
	if !strings.Contains(llm.prompts[0], "NOTE_START") {
		t.Error("extraction prompt lost the untrusted boundary")
	}
	if strings.Contains(llm.prompts[1], testDog.Description) {
		t.Error("generation prompt must never see the raw description")
	}
}

func TestCompileDropsUnverifiableFacts(t *testing.T) {
	invented := `{"facts":["Loves tennis balls and belly rubs","Was abused by a previous owner"]}`
	llm := &fakeLLM{responses: []string{invented, goodSheetWithoutF7F8(t)}}
	sheet, err := newTestCompiler(t, llm).Compile(context.Background(), testDog)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range sheet.Facts {
		if strings.Contains(f.Value, "abused") {
			t.Fatal("an invented fact survived verification")
		}
	}
}

// goodSheetWithoutF7F8 rewrites citations for the run where only one
// description fact survives, so ids stop at f6.
func goodSheetWithoutF7F8(t *testing.T) string {
	t.Helper()
	s := strings.ReplaceAll(goodSheet, `"cites": ["f7"]`, `"cites": ["f6"]`)
	s = strings.ReplaceAll(s, `"cites": ["f8"]`, `"cites": ["f6"]`)
	s = strings.ReplaceAll(s, `"fears": [{"value": "loud rumbles from the sky", "quote": "Scared of thunder, hides under the bed", "cites": ["f6"]}]`, `"fears": []`)
	return s
}

func TestCompileRetriesOnUngroundedSheet(t *testing.T) {
	ungrounded := strings.ReplaceAll(goodSheet, `"cites": ["f8"]`, `"cites": ["f99"]`)
	llm := &fakeLLM{responses: []string{goodExtraction, ungrounded, goodSheet}}
	sheet, err := newTestCompiler(t, llm).Compile(context.Background(), testDog)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.Default {
		t.Fatal("retry should have produced a real sheet")
	}
	if llm.calls != 3 {
		t.Errorf("calls = %d, want 3 (extract, bad sheet, good sheet)", llm.calls)
	}
}

func TestCompileFallsBackToDefaultOnAPIError(t *testing.T) {
	llm := &fakeLLM{errs: []error{&gemini.APIError{Status: 429, Detail: "quota"}}}
	sheet, err := newTestCompiler(t, llm).Compile(context.Background(), testDog)
	var degraded *Degraded
	if !errors.As(err, &degraded) {
		t.Fatalf("want a Degraded error naming the cause, got %v", err)
	}
	var apiErr *gemini.APIError
	if !errors.As(degraded, &apiErr) || apiErr.Status != 429 {
		t.Errorf("cause should unwrap to the api error, got %v", degraded.Cause)
	}
	if sheet == nil || !sheet.Default {
		t.Fatal("api failure must still serve the default sheet")
	}
	if len(sheet.Fears) != 0 {
		t.Error("default sheet must not carry fears")
	}
	if sheet.RadioSeed.Value == "" {
		t.Error("default sheet needs a radio seed")
	}
}

func TestCompilePropagatesCancellation(t *testing.T) {
	llm := &fakeLLM{errs: []error{context.Canceled}}
	sheet, err := newTestCompiler(t, llm).Compile(context.Background(), testDog)
	if sheet != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation must not be papered over with a default sheet, got sheet %v err %v", sheet, err)
	}
}

func TestCompileUsesCache(t *testing.T) {
	cache := NewCache(t.TempDir())
	llm := &fakeLLM{responses: []string{goodExtraction, goodSheet}}
	c := NewCompiler(llm, cache)

	first, err := c.Compile(context.Background(), testDog)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Compile(context.Background(), testDog)
	if err != nil {
		t.Fatal(err)
	}
	if llm.calls != 2 {
		t.Errorf("second compile must come from cache, calls = %d", llm.calls)
	}
	if second.AnimalID != first.AnimalID || len(second.Facts) != len(first.Facts) {
		t.Error("cached sheet does not match the original")
	}

	changed := testDog
	changed.Description = changed.Description + " New note line."
	llm.responses = append(llm.responses, goodExtraction, goodSheet)
	if _, err := c.Compile(context.Background(), changed); err != nil {
		t.Fatal(err)
	}
	if llm.calls != 4 {
		t.Errorf("a changed description must recompile, calls = %d", llm.calls)
	}
}

func TestDefaultSheetNeverCached(t *testing.T) {
	cache := NewCache(t.TempDir())
	llm := &fakeLLM{errs: []error{&gemini.APIError{Status: 500, Detail: "down"}}}
	c := NewCompiler(llm, cache)
	sheet, err := c.Compile(context.Background(), testDog)
	var degraded *Degraded
	if sheet == nil || !errors.As(err, &degraded) {
		t.Fatalf("want default sheet with Degraded error, got %v", err)
	}
	if _, ok := cache.Get(testDog); ok {
		t.Fatal("default sheets must not be cached, a later run should heal")
	}
}
