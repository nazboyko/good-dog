package dogsheet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/gemini"
)

// generator is the slice of the gemini client the compiler needs,
// tests script it without any network.
type generator interface {
	GenerateJSON(ctx context.Context, prompt string, opts gemini.Options) ([]byte, error)
}

type Compiler struct {
	llm   generator
	cache *Cache
}

func NewCompiler(llm generator, cache *Cache) *Compiler {
	return &Compiler{llm: llm, cache: cache}
}

// Compile runs two step grounding for one dog. Model failures never
// crash a session: the caller gets the neutral default sheet instead.
func (c *Compiler) Compile(ctx context.Context, a animal.Animal) (*DogSheet, error) {
	if sheet, ok := c.cache.Get(a); ok {
		return sheet, nil
	}

	facts := StructuredFacts(a)
	extracted, err := c.extract(ctx, a.Description)
	if err != nil {
		return c.fallback(a, facts, fmt.Errorf("extraction: %w", err))
	}
	kept, dropped := VerifiedDescriptionFacts(extracted, a.Description, len(facts), a.RetrievedAt)
	if len(dropped) > 0 {
		log.Printf("dog sheet %s: dropped %d extracted facts, not verbatim or adoption history", a.ID, len(dropped))
	}
	facts = append(facts, kept...)

	sheet, err := c.generate(ctx, a, facts)
	if err != nil {
		return c.fallback(a, facts, fmt.Errorf("generation: %w", err))
	}
	if err := c.cache.Put(a, sheet); err != nil {
		log.Printf("dog sheet %s: cache write failed: %v", a.ID, err)
	}
	return sheet, nil
}

// extract asks for verbatim quotes, retrying once on bad output.
func (c *Compiler) extract(ctx context.Context, description string) ([]string, error) {
	prompt := fmt.Sprintf(extractFactsPrompt, neutralizeSentinel(description))
	opts := gemini.Options{Temperature: 0.1, MaxOutputTokens: 1024, ResponseSchema: extractFactsSchema, DisableThinking: true}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := c.llm.GenerateJSON(ctx, prompt, opts)
		if err != nil {
			return nil, err
		}
		var out struct {
			Facts []string `json:"facts"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			lastErr = fmt.Errorf("output not json: %w", err)
			continue
		}
		return out.Facts, nil
	}
	return nil, lastErr
}

// generate builds the sheet from facts only, retrying once when the
// output fails schema parsing or the grounding verifier.
func (c *Compiler) generate(ctx context.Context, a animal.Animal, facts []VerifiedFact) (*DogSheet, error) {
	prompt := fmt.Sprintf(generateSheetPrompt, factsBlock(facts))
	opts := gemini.Options{Temperature: 0.8, MaxOutputTokens: 1024, ResponseSchema: generateSheetSchema, DisableThinking: true}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := c.llm.GenerateJSON(ctx, prompt, opts)
		if err != nil {
			return nil, err
		}
		sheet, err := parseSheet(raw, a, facts)
		if err != nil {
			lastErr = err
			continue
		}
		if errs := VerifySheet(sheet, a.Description); len(errs) > 0 {
			lastErr = fmt.Errorf("verifier rejected sheet: %v", errs[0])
			continue
		}
		return sheet, nil
	}
	return nil, lastErr
}

// Degraded reports that the default sheet was served and why. Game
// callers log it and keep playing, evals inspect the cause and abort
// their run on quota exhaustion instead of reporting false failures.
type Degraded struct {
	Cause error
}

func (d *Degraded) Error() string { return "default sheet served: " + d.Cause.Error() }
func (d *Degraded) Unwrap() error { return d.Cause }

// fallback serves the neutral default on the typed api error path and
// on unverifiable output, and only fails hard on context cancellation.
// A non nil sheet together with a Degraded error is a playable result.
func (c *Compiler) fallback(a animal.Animal, facts []VerifiedFact, cause error) (*DogSheet, error) {
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return nil, cause
	}
	var apiErr *gemini.APIError
	if errors.As(cause, &apiErr) {
		log.Printf("dog sheet %s: gemini status %d, serving default sheet", a.ID, apiErr.Status)
	} else {
		log.Printf("dog sheet %s: %v, serving default sheet", a.ID, cause)
	}
	return DefaultSheet(a, facts), &Degraded{Cause: cause}
}

type citedValue struct {
	Value string   `json:"value"`
	Cites []string `json:"cites"`
}

type sheetPayload struct {
	Matrix struct {
		Energy           axisPayload `json:"energy"`
		Confidence       axisPayload `json:"confidence"`
		Sociability      axisPayload `json:"sociability"`
		Patience         axisPayload `json:"patience"`
		FoodDrive        axisPayload `json:"food_drive"`
		NoiseSensitivity axisPayload `json:"noise_sensitivity"`
	} `json:"matrix"`
	Fears []struct {
		Value string   `json:"value"`
		Quote string   `json:"quote"`
		Cites []string `json:"cites"`
	} `json:"fears"`
	Quirks    []citedValue `json:"quirks"`
	Movement  citedValue   `json:"movement"`
	Voice     citedValue   `json:"voice"`
	RadioSeed citedValue   `json:"radio_seed"`
}

type axisPayload struct {
	Level string   `json:"level"`
	Cites []string `json:"cites"`
}

func parseSheet(raw []byte, a animal.Animal, facts []VerifiedFact) (*DogSheet, error) {
	var p sheetPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("sheet not json: %w", err)
	}
	axis := func(ap axisPayload) Axis {
		return Axis{Level: Level(ap.Level), DerivedFrom: ap.Cites}
	}
	sheet := &DogSheet{
		AnimalID: a.ID,
		Facts:    facts,
		Matrix: Matrix{
			Energy:           axis(p.Matrix.Energy),
			Confidence:       axis(p.Matrix.Confidence),
			Sociability:      axis(p.Matrix.Sociability),
			Patience:         axis(p.Matrix.Patience),
			FoodDrive:        axis(p.Matrix.FoodDrive),
			NoiseSensitivity: axis(p.Matrix.NoiseSensitivity),
		},
		Movement:   neutralUnlessCited(p.Movement, neutralMovement, CategoryBehavior),
		Voice:      neutralUnlessCited(p.Voice, neutralVoice, CategorySensory),
		RadioSeed:  NarrativeInference{Value: p.RadioSeed.Value, DerivedFrom: p.RadioSeed.Cites, Category: CategoryTone},
		CompiledAt: time.Now().UTC(),
	}
	for _, f := range p.Fears {
		sheet.Fears = append(sheet.Fears, Fear{Value: f.Value, Quote: f.Quote, DerivedFrom: f.Cites})
	}
	for _, q := range p.Quirks {
		sheet.Quirks = append(sheet.Quirks, NarrativeInference{Value: q.Value, DerivedFrom: q.Cites, Category: CategoryBehavior})
	}
	return sheet, nil
}

// The neutral lines used whenever the model has no fact to stand on.
// The model was told to be plainly neutral with empty cites but writes
// flourishes instead, so uncited voice and movement snap to these.
const (
	neutralMovement = "settles into new places at an easy pace"
	neutralVoice    = "quiet until something matters"
)

func neutralUnlessCited(cv citedValue, neutral, category string) NarrativeInference {
	if len(cv.Cites) == 0 {
		return NarrativeInference{Value: neutral, Category: category}
	}
	return NarrativeInference{Value: cv.Value, DerivedFrom: cv.Cites, Category: category}
}

// DefaultSheet is the canned default from the tech stack rules: neutral,
// claims nothing beyond the structured facts, safe to play, never cached
// so a later run can heal.
func DefaultSheet(a animal.Animal, facts []VerifiedFact) *DogSheet {
	neutral := Axis{Level: LevelMedium}
	seedCites := []string{}
	for _, f := range facts {
		if f.Source == "field:name" || f.Source == "field:breed" || f.Source == "field:age_group" {
			seedCites = append(seedCites, f.ID)
		}
	}
	return &DogSheet{
		AnimalID: a.ID,
		Facts:    facts,
		Matrix: Matrix{
			Energy: neutral, Confidence: neutral, Sociability: neutral,
			Patience: neutral, FoodDrive: neutral, NoiseSensitivity: neutral,
		},
		Movement:   NarrativeInference{Value: neutralMovement, Category: CategoryBehavior},
		Voice:      NarrativeInference{Value: neutralVoice, Category: CategorySensory},
		RadioSeed:  NarrativeInference{Value: fmt.Sprintf("%s is here tonight, and that is a good thing.", a.Name), DerivedFrom: seedCites, Category: CategoryTone},
		Default:    true,
		CompiledAt: time.Now().UTC(),
	}
}
