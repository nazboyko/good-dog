package dogsheet

import (
	"fmt"
	"strings"
)

// The verifier is the engine owning truth: it mechanically rejects any
// sheet whose claims cannot be traced to verified facts. The forbidden
// lists mirror the hallucination classes in the verified-animal-data
// skill. A term in these classes is allowed when the listing itself
// uses it, quoting reality is never a violation.
var forbiddenIfInvented = map[string][]string{
	"trauma":     {"abused", "abuse", "beaten", "neglected", "mistreated", "trauma", "traumatized", "abandoned", "surrender", "gave up", "given up", "left behind"},
	"medical":    {"arthritis", "cancer", "sick", "illness", "diagnosed", "medication", "meds", "blind", "deaf", "heartworm", "injury", "injured", "surgery"},
	"aggression": {"aggressive", "aggression", "reactive", "reactivity", "bites", "bite history", "has bitten", "attacks", "dangerous"},
	"shelter":    {"staff", "volunteer", "came to us", "been here", "been waiting", "waiting for"},
}

// These classes are forbidden in generated text no matter what the
// listing says: adopted appears only with ADOPTED_CONFIRMED, and fake
// urgency is never allowed, per the hard content rules.
var forbiddenAlways = map[string][]string{
	"adoption": {"adopt", "forever home", "found a home"},
	"urgency":  {"euthan", "put down", "last chance", "running out of time", "time is running out"},
}

// containsStem is raw substring matching on normalized text, so the
// stem adopt catches adopted and adoption. Quote checks use word
// boundaries instead. Over matching here only costs a retry, letting
// a forbidden claim through costs the truth.
func containsStem(stem, text string) bool {
	s := normalizeForMatch(stem)
	if s == "" {
		return false
	}
	return strings.Contains(normalizeForMatch(text), s)
}

// VerifySheet returns every grounding violation it can prove. An empty
// slice means the sheet may exist.
func VerifySheet(sheet *DogSheet, description string) []error {
	var errs []error
	factIDs := map[string]bool{}
	factText := &strings.Builder{}
	for _, f := range sheet.Facts {
		factIDs[f.ID] = true
		factText.WriteString(f.Value)
		factText.WriteString(" ")
		// re-checked here so even cached sheets can never carry a
		// description fact that is not a verbatim quote
		if f.Source == "description" && !QuoteInText(f.Value, description) {
			errs = append(errs, fmt.Errorf("fact %s is not a quote from the description: %q", f.ID, f.Value))
		}
	}
	sourceText := description + " " + factText.String()

	checkCites := func(where string, cites []string) {
		for _, id := range cites {
			if !factIDs[id] {
				errs = append(errs, fmt.Errorf("%s cites %s which is not a verified fact", where, id))
			}
		}
	}
	checkForbidden := func(where, text string) {
		for class, terms := range forbiddenIfInvented {
			for _, term := range terms {
				if containsStem(term, text) && !containsStem(term, sourceText) {
					errs = append(errs, fmt.Errorf("%s invents %s content, contains %q", where, class, term))
				}
			}
		}
		for class, terms := range forbiddenAlways {
			for _, term := range terms {
				if containsStem(term, text) {
					errs = append(errs, fmt.Errorf("%s contains %s language %q, forbidden regardless of the listing", where, class, term))
				}
			}
		}
	}

	axes := map[string]Axis{
		"energy": sheet.Matrix.Energy, "confidence": sheet.Matrix.Confidence,
		"sociability": sheet.Matrix.Sociability, "patience": sheet.Matrix.Patience,
		"food_drive": sheet.Matrix.FoodDrive, "noise_sensitivity": sheet.Matrix.NoiseSensitivity,
	}
	for name, axis := range axes {
		checkCites("matrix."+name, axis.DerivedFrom)
		if len(axis.DerivedFrom) == 0 && axis.Level != LevelMedium {
			errs = append(errs, fmt.Errorf("matrix.%s is %s without citing any fact", name, axis.Level))
		}
		if axis.Level != LevelLow && axis.Level != LevelMedium && axis.Level != LevelHigh {
			errs = append(errs, fmt.Errorf("matrix.%s has unknown level %q", name, axis.Level))
		}
	}

	for i, fear := range sheet.Fears {
		where := fmt.Sprintf("fears[%d]", i)
		checkCites(where, fear.DerivedFrom)
		if len(fear.DerivedFrom) == 0 {
			errs = append(errs, fmt.Errorf("%s cites nothing", where))
		}
		if !QuoteInText(fear.Quote, description) {
			errs = append(errs, fmt.Errorf("%s quote %q is not in the description", where, fear.Quote))
		}
		checkForbidden(where, fear.Value)
	}

	inferences := map[string]NarrativeInference{
		"movement": sheet.Movement, "voice": sheet.Voice, "radio_seed": sheet.RadioSeed,
	}
	for i, q := range sheet.Quirks {
		inferences[fmt.Sprintf("quirks[%d]", i)] = q
	}
	for where, inf := range inferences {
		checkCites(where, inf.DerivedFrom)
		checkForbidden(where, inf.Value)
		if inf.Value == "" {
			errs = append(errs, fmt.Errorf("%s is empty", where))
		}
		// movement and voice may be plainly neutral with empty cites,
		// quirks and the radio seed always stand on facts
		if len(inf.DerivedFrom) == 0 && (strings.HasPrefix(where, "quirks") || where == "radio_seed") {
			errs = append(errs, fmt.Errorf("%s cites nothing", where))
		}
	}
	return errs
}
