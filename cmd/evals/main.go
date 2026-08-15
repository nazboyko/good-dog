// Command evals runs the grounding eval dataset against the live model,
// or compiles fixture dogs with -fixture. Red evals block prompt changes.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/config"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/gemini"
)

type expectations struct {
	NoFears        bool              `json:"no_fears"`
	FearExpected   bool              `json:"fear_expected"`
	AllAxesDefault bool              `json:"all_axes_default"`
	MinFacts       int               `json:"min_facts"`
	MustQuote      []string          `json:"must_quote"`
	CanaryAbsent   []string          `json:"canary_absent"`
	AxisLevels     map[string]string `json:"axis_levels"`
}

type profile struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Breed           string       `json:"breed"`
	AgeGroup        string       `json:"age_group"`
	AgeText         string       `json:"age_text"`
	Sex             string       `json:"sex"`
	Size            string       `json:"size"`
	WeightText      string       `json:"weight_text"`
	DescriptionHTML string       `json:"description_html"`
	Expect          expectations `json:"expect"`
}

func main() {
	profilesPath := flag.String("profiles", "evals/profiles.json", "eval dataset")
	fixtureIDs := flag.String("fixture", "", "comma separated fixture dog ids to compile and print instead of running evals")
	only := flag.String("only", "", "comma separated profile ids, everything else is skipped")
	show := flag.Bool("show", false, "print the full sheet for every failing profile")
	flag.Parse()

	if err := config.LoadDotEnv(".env"); err != nil {
		fatal("load .env: %v", err)
	}
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		fatal("GEMINI_API_KEY not set")
	}
	llm := gemini.New(key)

	if *fixtureIDs != "" {
		compileFixtures(llm, strings.Split(*fixtureIDs, ","))
		return
	}
	runEvals(llm, *profilesPath, *only, *show)
}

// compileFixtures uses the real sheet cache, evals never do.
func compileFixtures(llm *gemini.Client, ids []string) {
	provider, err := animal.NewFixtureProvider("fixtures/dogs.json")
	if err != nil {
		fatal("%v", err)
	}
	compiler := dogsheet.NewCompiler(llm, dogsheet.NewCache("cache/sheets"))
	for _, id := range ids {
		a, err := provider.GetAnimal(context.Background(), strings.TrimSpace(id))
		if err != nil {
			fatal("%v", err)
		}
		sheet, err := compiler.Compile(context.Background(), *a)
		if err != nil {
			fatal("compile %s: %v", id, err)
		}
		out, _ := json.MarshalIndent(sheet, "", "  ")
		fmt.Printf("=== dog sheet %s (%s) ===\n%s\n", a.Name, a.ID, out)
	}
}

// errQuotaExhausted aborts the whole run: a starved model proves nothing
// about grounding, and burning retries on it makes the starvation worse.
var errQuotaExhausted = fmt.Errorf("quota exhausted")

func runEvals(llm *gemini.Client, path, only string, show bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal("read profiles: %v", err)
	}
	var file struct {
		Profiles []profile `json:"profiles"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		fatal("parse profiles: %v", err)
	}
	selected := file.Profiles
	if only != "" {
		wanted := map[string]bool{}
		for _, id := range strings.Split(only, ",") {
			wanted[strings.TrimSpace(id)] = true
		}
		selected = nil
		for _, p := range file.Profiles {
			if wanted[p.ID] {
				selected = append(selected, p)
			}
		}
		if len(selected) == 0 {
			fatal("-only matched no profiles, a typo would read as a green run")
		}
	}

	cacheDir, err := os.MkdirTemp("", "eval-sheets")
	if err != nil {
		fatal("%v", err)
	}
	defer os.RemoveAll(cacheDir)
	compiler := dogsheet.NewCompiler(llm, dogsheet.NewCache(cacheDir))

	failed := 0
	start := time.Now()
	for i, p := range selected {
		problems, sheet, err := runProfile(compiler, p)
		if err != nil {
			fmt.Printf("\nquota exhausted at profile %s, %d of %d done, aborting the run\n", p.ID, i, len(selected))
			fmt.Println("this is starvation, not a grounding failure, rerun when the window resets")
			os.Exit(2)
		}
		if len(problems) == 0 {
			fmt.Printf("PASS %s\n", p.ID)
			continue
		}
		failed++
		for _, problem := range problems {
			fmt.Printf("FAIL %s: %s\n", p.ID, problem)
		}
		if show && sheet != nil {
			out, _ := json.MarshalIndent(sheet, "", "  ")
			fmt.Printf("--- sheet for %s ---\n%s\n", p.ID, out)
		}
	}
	fmt.Printf("\n%d profiles, %d failed, %s\n", len(selected), failed, time.Since(start).Round(time.Second))
	if failed > 0 {
		os.Exit(1)
	}
}

func runProfile(compiler *dogsheet.Compiler, p profile) ([]string, *dogsheet.DogSheet, error) {
	a := animal.Animal{
		ID: "eval-" + p.ID, Name: p.Name, Breed: p.Breed, AgeGroup: p.AgeGroup,
		AgeText: p.AgeText, Sex: p.Sex, Size: p.Size, WeightText: p.WeightText,
		Description: animal.SanitizeDescription(p.DescriptionHTML),
		RetrievedAt: time.Now().UTC(),
	}
	sheet, err := compiler.Compile(context.Background(), a)
	var degraded *dogsheet.Degraded
	if errors.As(err, &degraded) {
		var apiErr *gemini.APIError
		if errors.As(degraded, &apiErr) && apiErr.Status == 429 {
			return nil, nil, errQuotaExhausted
		}
		return []string{fmt.Sprintf("served the default sheet: %v", degraded.Cause)}, sheet, nil
	}
	if err != nil {
		return []string{fmt.Sprintf("compile error: %v", err)}, nil, nil
	}

	var problems []string
	for _, verr := range dogsheet.VerifySheet(sheet, a.Description) {
		problems = append(problems, "verifier: "+verr.Error())
	}
	// quote verification lives inside VerifySheet now, count only
	descFacts := 0
	for _, f := range sheet.Facts {
		if f.Source == "description" {
			descFacts++
		}
	}

	e := p.Expect
	if e.NoFears && len(sheet.Fears) > 0 {
		problems = append(problems, fmt.Sprintf("expected no fears, got %d", len(sheet.Fears)))
	}
	if e.FearExpected && len(sheet.Fears) == 0 {
		problems = append(problems, "expected a fear grounded in the description, got none")
	}
	if e.AllAxesDefault {
		for name, axis := range axesOf(sheet) {
			if axis.Level != dogsheet.LevelMedium || len(axis.DerivedFrom) > 0 {
				problems = append(problems, fmt.Sprintf("axis %s should be default medium, got %s cited %v", name, axis.Level, axis.DerivedFrom))
			}
		}
	}
	for name, want := range e.AxisLevels {
		axis, ok := axesOf(sheet)[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("expectations name axis %q, which does not exist", name))
			continue
		}
		if string(axis.Level) != want {
			problems = append(problems, fmt.Sprintf("axis %s is %s, expected %s", name, axis.Level, want))
		}
	}
	if descFacts < e.MinFacts {
		problems = append(problems, fmt.Sprintf("kept %d description facts, expected at least %d", descFacts, e.MinFacts))
	}
	for _, quote := range e.MustQuote {
		found := false
		for _, f := range sheet.Facts {
			if dogsheet.QuoteInText(quote, f.Value) {
				found = true
				break
			}
		}
		if !found {
			problems = append(problems, fmt.Sprintf("no kept fact mentions %q", quote))
		}
	}
	for _, canary := range e.CanaryAbsent {
		for where, text := range allSheetText(sheet) {
			if dogsheet.QuoteInText(canary, text) {
				problems = append(problems, fmt.Sprintf("canary %q leaked into %s", canary, where))
			}
		}
	}
	return problems, sheet, nil
}

func axesOf(s *dogsheet.DogSheet) map[string]dogsheet.Axis {
	return map[string]dogsheet.Axis{
		"energy": s.Matrix.Energy, "confidence": s.Matrix.Confidence,
		"sociability": s.Matrix.Sociability, "patience": s.Matrix.Patience,
		"food_drive": s.Matrix.FoodDrive, "noise_sensitivity": s.Matrix.NoiseSensitivity,
	}
}

func allSheetText(s *dogsheet.DogSheet) map[string]string {
	out := map[string]string{
		"movement": s.Movement.Value, "voice": s.Voice.Value, "radio_seed": s.RadioSeed.Value,
	}
	for i, q := range s.Quirks {
		out[fmt.Sprintf("quirks[%d]", i)] = q.Value
	}
	for i, f := range s.Fears {
		out[fmt.Sprintf("fears[%d]", i)] = f.Value + " " + f.Quote
	}
	return out
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
