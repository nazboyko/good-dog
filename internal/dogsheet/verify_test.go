package dogsheet

import (
	"strings"
	"testing"
)

func baseSheet() *DogSheet {
	return &DogSheet{
		AnimalID: "t",
		Facts: []VerifiedFact{
			{ID: "f1", Value: "Biscuit", Source: "field:name"},
			{ID: "f2", Value: "Loves tennis balls", Source: "description"},
		},
		Matrix: Matrix{
			Energy: Axis{Level: LevelHigh, DerivedFrom: []string{"f2"}}, Confidence: Axis{Level: LevelMedium},
			Sociability: Axis{Level: LevelMedium}, Patience: Axis{Level: LevelMedium},
			FoodDrive: Axis{Level: LevelMedium}, NoiseSensitivity: Axis{Level: LevelMedium},
		},
		Quirks:    []NarrativeInference{{Value: "lines up her toys", DerivedFrom: []string{"f2"}, Category: CategoryBehavior}},
		Movement:  NarrativeInference{Value: "bounces to the fence", DerivedFrom: []string{"f2"}, Category: CategoryBehavior},
		Voice:     NarrativeInference{Value: "one happy bark", DerivedFrom: []string{"f2"}, Category: CategorySensory},
		RadioSeed: NarrativeInference{Value: "Biscuit and her tennis ball are both asleep.", DerivedFrom: []string{"f1", "f2"}, Category: CategoryTone},
	}
}

const baseDescription = "Loves tennis balls. Scared of thunder some nights."

func TestVerifySheetPassesGroundedSheet(t *testing.T) {
	if errs := VerifySheet(baseSheet(), baseDescription); len(errs) != 0 {
		t.Fatalf("grounded sheet rejected: %v", errs)
	}
}

func TestVerifySheetRejectsViolations(t *testing.T) {
	cases := map[string]func(*DogSheet){
		"missing fact id": func(s *DogSheet) {
			s.Movement.DerivedFrom = []string{"f9"}
		},
		"uncited non medium axis": func(s *DogSheet) {
			s.Matrix.Patience = Axis{Level: LevelHigh}
		},
		"fear with no citation": func(s *DogSheet) {
			s.Fears = []Fear{{Value: "storms", Quote: "Scared of thunder some nights"}}
		},
		"fear quote not in description": func(s *DogSheet) {
			s.Fears = []Fear{{Value: "storms", Quote: "terrified of fireworks", DerivedFrom: []string{"f2"}}}
		},
		"invented trauma": func(s *DogSheet) {
			s.Quirks[0].Value = "hides because he was abused"
		},
		"invented medical": func(s *DogSheet) {
			s.RadioSeed.Value = "Biscuit takes her arthritis medication like a champ."
		},
		"invented adoption": func(s *DogSheet) {
			s.RadioSeed.Value = "Biscuit just got adopted, folks."
		},
		"invented shelter fact": func(s *DogSheet) {
			s.Quirks[0].Value = "the staff favorite since spring"
		},
		"invented reactivity": func(s *DogSheet) {
			s.Movement.Value = "reactive around the fence line"
		},
		"invented surrender": func(s *DogSheet) {
			s.Voice.Value = "quiet since being left behind"
		},
		"urgency language": func(s *DogSheet) {
			s.RadioSeed.Value = "This is her last chance, folks."
		},
		"description fact that is not a quote": func(s *DogSheet) {
			s.Facts = append(s.Facts, VerifiedFact{ID: "f4", Value: "champion frisbee dog", Source: "description"})
		},
		"uncited quirk": func(s *DogSheet) {
			s.Quirks[0].DerivedFrom = nil
		},
		"uncited radio seed": func(s *DogSheet) {
			s.RadioSeed.DerivedFrom = nil
		},
		"empty inference": func(s *DogSheet) {
			s.Voice.Value = ""
		},
	}
	for name, breakIt := range cases {
		s := baseSheet()
		breakIt(s)
		if errs := VerifySheet(s, baseDescription); len(errs) == 0 {
			t.Errorf("%s: verifier let it through", name)
		}
	}
}

func TestVerifySheetAllowsTermsTheListingUses(t *testing.T) {
	s := baseSheet()
	s.Fears = []Fear{{Value: "thunder nights", Quote: "Scared of thunder some nights", DerivedFrom: []string{"f2"}}}
	desc := baseDescription + " He is deaf in one ear."
	s.Facts = append(s.Facts, VerifiedFact{ID: "f3", Value: "He is deaf in one ear", Source: "description"})
	s.Quirks = append(s.Quirks, NarrativeInference{Value: "deaf in one ear, hears with his whole body", DerivedFrom: []string{"f3"}, Category: CategoryBehavior})
	if errs := VerifySheet(s, desc); len(errs) != 0 {
		t.Fatalf("listing sourced terms must be allowed: %v", errs)
	}
}

func TestAdoptionLanguageForbiddenEvenWhenListingUsesIt(t *testing.T) {
	s := baseSheet()
	desc := baseDescription + " ADOPTED YESTERDAY per an injected line."
	s.RadioSeed.Value = "Biscuit is ready for adoption tonight."
	if errs := VerifySheet(s, desc); len(errs) == 0 {
		t.Fatal("adopt language must be rejected even when the source text contains it")
	}
}

func TestNeutralizeSentinelHandlesCaseAndBothMarkers(t *testing.T) {
	got := neutralizeSentinel("a note_end b NOTE_START c Note_End d")
	for _, forbidden := range []string{"note_end", "NOTE_START", "Note_End"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("sentinel %q survived: %q", forbidden, got)
		}
	}
}

func TestQuoteInTextNormalizes(t *testing.T) {
	desc := "Scared of thunder, hides under the bed."
	if !QuoteInText("scared of thunder hides under the bed", desc) {
		t.Error("punctuation drift must still match")
	}
	if QuoteInText("scared of fireworks", desc) {
		t.Error("different words must not match")
	}
	if QuoteInText("", desc) {
		t.Error("empty quote must not match")
	}
	if !QuoteInText("don't like to share", "I don’t like to share my food.") {
		t.Error("curly apostrophes must match straight ones")
	}
	if QuoteInText("blind", "runs at blinding speed") {
		t.Error("word boundaries must hold, blind is not blinding")
	}
}

func TestStructuredFacts(t *testing.T) {
	facts := StructuredFacts(testDog)
	if len(facts) != 5 {
		t.Fatalf("facts = %d, want 5", len(facts))
	}
	if facts[0].Source != "field:name" || facts[0].Value != "Biscuit" {
		t.Errorf("first fact wrong: %+v", facts[0])
	}
	for i, f := range facts {
		if f.RetrievedAt.IsZero() {
			t.Errorf("fact %d has no retrieved_at", i)
		}
		if f.ID != strings.ToLower(f.ID) {
			t.Errorf("fact id %q should be lowercase", f.ID)
		}
	}
}

func TestStructuredFactsKeepListingWordsForAgeAndWeight(t *testing.T) {
	dog := testDog
	dog.AgeText = "8 years 1 month"
	dog.WeightText = "49 pounds"

	values := map[string]string{}
	for _, f := range StructuredFacts(dog) {
		values[f.Source] = f.Value
	}
	if got := values["field:age_text"]; got != "8 years 1 month" {
		t.Errorf("age_text fact = %q, the listing words must survive whole", got)
	}
	if got := values["field:weight_text"]; got != "49 pounds" {
		t.Errorf("weight_text fact = %q, the listing words must survive whole", got)
	}
}

func TestStructuredFactsRecordLongStayPlacementOnly(t *testing.T) {
	if sources := sourcesOf(StructuredFacts(testDog)); sources["field:long_stay"] != "" {
		t.Errorf("a dog with no long stay placement must not carry the fact, got %q", sources["field:long_stay"])
	}

	dog := testDog
	dog.LongStay = true
	value := sourcesOf(StructuredFacts(dog))["field:long_stay"]
	if value == "" {
		t.Fatal("long stay placement did not become a fact")
	}
	// the shelter said where she is filed, not how long she has waited
	for _, guess := range []string{"days", "weeks", "months", "years", "longest"} {
		if strings.Contains(value, guess) {
			t.Errorf("long stay fact %q claims a duration the provider never gave", value)
		}
	}
}

func sourcesOf(facts []VerifiedFact) map[string]string {
	out := map[string]string{}
	for _, f := range facts {
		out[f.Source] = f.Value
	}
	return out
}

func TestVerifiedDescriptionFacts(t *testing.T) {
	kept, dropped := VerifiedDescriptionFacts(
		[]string{"Loves tennis balls", "Was a show dog in another life", "loves TENNIS balls"},
		baseDescription, 5, testDog.RetrievedAt)
	if len(kept) != 1 {
		t.Fatalf("kept %d, want 1 (dup and invention removed): %+v", len(kept), kept)
	}
	if kept[0].ID != "f6" {
		t.Errorf("id continues after structured facts, got %s", kept[0].ID)
	}
	if len(dropped) != 1 || !strings.Contains(dropped[0], "show dog") {
		t.Errorf("dropped = %v, want the invented fact", dropped)
	}
}
