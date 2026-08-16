package session

import (
	"strings"
	"testing"
)

// The dog in the next kennel is nobody, and has to stay nobody.
//
// The another dog ending says somebody was adopted today. That is safe
// only because the dog it refers to is invented furniture: no name, no
// photo, no status, not bound to anything in the pool. The design doc
// wants neighbour kennels to hold real dogs from the pool one day. The
// day that ships, this line becomes an invented adoption status for a
// real animal, which is the one thing layer one forbids outright, and
// it will already be baked into a tuned ending nobody wants to touch.
//
// So the constraint is written down here rather than in a comment
// somebody can miss.
func TestTheNextKennelIsNeverARealDog(t *testing.T) {
	line := EndingLine(EndingAnotherDog, "Venus", "her")
	if !strings.Contains(line, "the dog in the next kennel") {
		t.Fatalf("the another dog ending changed shape, re-check this rule: %s", line)
	}
	// no name, no shelter, no listing: the only thing said about that
	// dog is where it was standing
	for _, bound := range []string{"Bella", "Keno", "Ruff Start", "Animal Humane", "shelterluv", "http"} {
		if strings.Contains(line, bound) {
			t.Errorf("the next kennel is bound to something real: %s", line)
		}
	}
	// and nothing in the line is about the dog the player was
	if strings.Contains(line, "Venus") {
		t.Errorf("the another dog ending says nothing about this dog: %s", line)
	}
}

// The design note that dropped the night from adoption day also dropped
// the consolation the design promised for this ending. Recorded so the
// next person reads it as a decision rather than a hole.
func TestTheAnotherDogEndingHasNoRadioConsolation(t *testing.T) {
	// day three is wake, scent, adoption: no night beat at all
	for _, b := range FullRun.LastDay {
		if b == BeatNight {
			t.Error("adoption day has a night again, so the radio consolation is back on the table")
		}
	}
}
