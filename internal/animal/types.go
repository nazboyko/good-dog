package animal

import (
	"fmt"
	"time"
)

// Status is the whole lifecycle the game is allowed to know about.
// The word adopted is only ever true with ADOPTED_CONFIRMED.
type Status string

const (
	StatusActive           Status = "ACTIVE"
	StatusRemovedUnknown   Status = "REMOVED_UNKNOWN"
	StatusTransferred      Status = "TRANSFERRED"
	StatusUnavailable      Status = "UNAVAILABLE"
	StatusAdoptedConfirmed Status = "ADOPTED_CONFIRMED"
)

// Playable is whether the game may still put a player inside this dog.
//
// Adopted dogs stay in. Deleting one would be a lie by omission and it
// would throw away the thing the game is for, which is that these are
// real animals whose situations change while nobody is looking. The
// reveal has a truthful line for every status here, so the only rule is
// that the game must never claim she is waiting when she is not.
func Playable(s Status) bool {
	switch s {
	case StatusActive, StatusAdoptedConfirmed, StatusRemovedUnknown, StatusTransferred, StatusUnavailable:
		return true
	}
	return false
}

func ValidStatus(s Status) bool {
	switch s {
	case StatusActive, StatusRemovedUnknown, StatusTransferred, StatusUnavailable, StatusAdoptedConfirmed:
		return true
	}
	return false
}

// The two ways a provider can tell us a dog is a long stay animal:
// where it filed the dog, or what the listing itself says.
const (
	LongStayPlacement = "placement"
	LongStaySentence  = "listing sentence"
)

// ValidLongStay keeps the flag and its evidence in step. A dog is filed
// as long stay for a reason we can name, or not at all.
func ValidLongStay(a Animal) error {
	if !a.LongStay {
		if a.LongStayEvidence != "" {
			return fmt.Errorf("has long_stay_evidence %q without long_stay", a.LongStayEvidence)
		}
		return nil
	}
	switch a.LongStayEvidence {
	case LongStayPlacement, LongStaySentence:
		return nil
	case "":
		return fmt.Errorf("is long_stay with no evidence, use %q or %q", LongStayPlacement, LongStaySentence)
	default:
		return fmt.Errorf("has unknown long_stay_evidence %q", a.LongStayEvidence)
	}
}

// Animal is the normalized internal shape. Provider payloads stop at the
// adapter, the rest of the game only ever sees this.
type Animal struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Breed    string `json:"breed"`
	AgeGroup string `json:"age_group"`
	Sex      string `json:"sex"`
	Size     string `json:"size"`
	// AgeText and WeightText are the listing's own words, kept verbatim and
	// never parsed into numbers. Empty when the listing does not say.
	AgeText    string `json:"age_text,omitempty"`
	WeightText string `json:"weight_text,omitempty"`
	// LongStay is set when the provider files the dog as a long stay
	// animal. The claim is that filing, never a guess at how long.
	// LongStayEvidence records how we know, since the fact reads the
	// same either way and the source would otherwise be lost.
	LongStay         bool   `json:"long_stay,omitempty"`
	LongStayEvidence string `json:"long_stay_evidence,omitempty"`
	Description      string `json:"description"`
	PhotoURL         string `json:"photo_url"`
	PhotoLocal       string `json:"photo_local"`
	ListingURL       string `json:"listing_url"`
	OrgID            string `json:"org_id"`
	Status           Status `json:"status"`
	// AdoptedOn is the date the listing itself gives, copied as written,
	// for example "August 15, 2026". Only ever set with
	// ADOPTED_CONFIRMED, because it is the listing's claim and not ours.
	AdoptedOn   string    `json:"adopted_on,omitempty"`
	RetrievedAt time.Time `json:"retrieved_at"`
	// Synthetic marks shape examples, they must never reach a player
	Synthetic bool `json:"synthetic,omitempty"`
}

type Organization struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
	URL   string `json:"url"`
}
