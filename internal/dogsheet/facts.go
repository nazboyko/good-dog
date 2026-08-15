package dogsheet

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
)

// StructuredFacts converts listing fields to facts without any model:
// structured fields are facts by construction.
func StructuredFacts(a animal.Animal) []VerifiedFact {
	fields := []struct{ source, value string }{
		{"field:name", a.Name},
		{"field:breed", a.Breed},
		{"field:age_group", a.AgeGroup},
		{"field:age_text", a.AgeText},
		{"field:sex", a.Sex},
		{"field:size", a.Size},
		{"field:weight_text", a.WeightText},
	}
	// the fact is where the provider filed her, not a length of time
	if a.LongStay {
		fields = append(fields, struct{ source, value string }{
			"field:long_stay", "listed among this organization's long stay animals",
		})
	}
	var facts []VerifiedFact
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		facts = append(facts, VerifiedFact{
			ID:          fmt.Sprintf("f%d", len(facts)+1),
			Value:       f.value,
			Source:      f.source,
			RetrievedAt: a.RetrievedAt,
		})
	}
	return facts
}

var punctuation = regexp.MustCompile(`[^\p{L}\p{N} ]+`)

// normalizeForMatch makes quote checks robust to case and punctuation
// drift without letting invented words through.
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = punctuation.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

// QuoteInText reports whether the quote appears on word boundaries in
// the text after both sides are normalized. Empty quotes never match,
// and blind must never match blinding.
func QuoteInText(quote, text string) bool {
	q := normalizeForMatch(quote)
	if q == "" {
		return false
	}
	return strings.Contains(" "+normalizeForMatch(text)+" ", " "+q+" ")
}

// VerifiedDescriptionFacts keeps only extracted facts that really are
// quotes from the description, dropped facts are returned for the log.
// Kept facts get ids continuing after the structured facts.
func VerifiedDescriptionFacts(extracted []string, description string, startAt int, retrievedAt time.Time) (kept []VerifiedFact, dropped []string) {
	seen := map[string]bool{}
	for _, value := range extracted {
		value = strings.TrimSpace(value)
		norm := normalizeForMatch(value)
		if norm == "" || seen[norm] {
			continue
		}
		seen[norm] = true
		if !QuoteInText(value, description) {
			dropped = append(dropped, value)
			continue
		}
		// a real quote can still carry adoption history, the word adopted
		// belongs to ADOPTED_CONFIRMED and nowhere else in the game
		if strings.Contains(norm, "adopt") {
			dropped = append(dropped, value)
			continue
		}
		kept = append(kept, VerifiedFact{
			ID:          fmt.Sprintf("f%d", startAt+len(kept)+1),
			Value:       value,
			Source:      "description",
			RetrievedAt: retrievedAt,
		})
	}
	return kept, dropped
}
