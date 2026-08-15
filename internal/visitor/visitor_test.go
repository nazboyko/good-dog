package visitor

import (
	"strings"
	"testing"
)

func TestEveryArchetypeHasAnOpinionAboutEverySignal(t *testing.T) {
	for _, a := range Archetypes {
		if len(a.Prefers) != len(Signals) {
			t.Errorf("%s has %d preferences, want %d", a.ID, len(a.Prefers), len(Signals))
		}
		for _, s := range Signals {
			if _, ok := a.Prefers[s]; !ok {
				// a missing entry would read as indifference by accident
				t.Errorf("%s has no opinion about %s", a.ID, s)
			}
		}
	}
}

// The band language is the whole tutorial. Each band must say a
// different thing a body does, or a player cannot read the room.
func TestEachBandHasItsOwnBodyLanguage(t *testing.T) {
	for _, a := range Archetypes {
		seen := map[string]Band{}
		for _, b := range Bands {
			line := BodyLanguage(a, b)
			if line == "" {
				t.Fatalf("%s has no body language for band %s", a.ID, b)
			}
			if other, dup := seen[line]; dup {
				t.Errorf("%s: bands %s and %s share a line: %q", a.ID, other, b, line)
			}
			seen[line] = b
			if !strings.HasSuffix(line, ".") || strings.Contains(line, "—") {
				t.Errorf("%s: band %s line is not plain copy: %q", a.ID, b, line)
			}
		}
		if len(seen) != len(Bands) {
			t.Errorf("%s: %d distinct lines for %d bands", a.ID, len(seen), len(Bands))
		}
	}
}

func TestBandsAreGraded(t *testing.T) {
	// more comfort never reads as less, and the whole range is used
	order := map[Band]int{}
	for i, b := range Bands {
		order[b] = i
	}
	last := -1
	for comfort := -4; comfort <= 4; comfort++ {
		got := order[BandFor(comfort)]
		if got < last {
			t.Errorf("comfort %d graded down to %s", comfort, BandFor(comfort))
		}
		last = got
	}
	if BandFor(-4) != BandDrifting || BandFor(4) != BandClose || BandFor(0) != BandWatching {
		t.Error("the ends and the middle of the scale must be reachable")
	}
}

// Honest matching: a visitor who came for another dog can never take
// this one home, no matter what the player does. That is not a failure
// state, it is the truth of a shelter afternoon.
func TestTheImpossibleVisitorCanNeverChooseButCanAlwaysPartWell(t *testing.T) {
	best := OutcomeMovedOn
	for _, s := range Signals {
		r := Meet(HereForAnother, s)
		if r.Outcome == OutcomeAsked {
			t.Errorf("signal %s made an impossible visitor choose", s)
		}
		if r.Outcome == OutcomeParted {
			best = OutcomeParted
		}
	}
	if best != OutcomeParted {
		t.Error("parting well must be reachable, or the scene is not worth playing")
	}
	// and the visitor who is looking can be reached
	reached := false
	for _, s := range Signals {
		if Meet(QuietSeeker, s).Outcome == OutcomeAsked {
			reached = true
		}
	}
	if !reached {
		t.Error("a visitor who is looking must be reachable by some signal")
	}
}

// The game never says wrong. Every line a player can read at parting
// says what happened, not how they did.
func TestPartingCopyIsConsequenceNeverFailure(t *testing.T) {
	// the arrival, the body and the parting all speak in the present,
	// so a player never has to wonder if the visitor is still there
	for _, a := range Archetypes {
		for _, o := range []Outcome{OutcomeAsked, OutcomeParted, OutcomeMovedOn} {
			line := Parting(a, o, BandWarming)
			for _, pastTense := range []string{"stopped at", "said goodbye", "moved on down", "stayed at"} {
				if strings.Contains(line, pastTense) {
					t.Errorf("%s parting for %s slips into past tense: %s", a.ID, o, line)
				}
			}
		}
	}

	blame := []string{"wrong", "fail", "should have", "too bad", "unfortunately", "sorry",
		"missed", "lost", "mistake", "better luck", "try again"}
	for _, a := range Archetypes {
		for _, o := range []Outcome{OutcomeAsked, OutcomeParted, OutcomeMovedOn} {
			for _, b := range Bands {
				line := Parting(a, o, b)
				if line == "" {
					t.Fatalf("%s has no parting line for %s", a.ID, o)
				}
				lower := strings.ToLower(line)
				for _, word := range blame {
					if strings.Contains(lower, word) {
						t.Errorf("%s parting for %s blames the player with %q: %s", a.ID, o, word, line)
					}
				}
				if strings.Contains(line, "—") || strings.Contains(line, "!") {
					t.Errorf("%s parting for %s is not the house voice: %s", a.ID, o, line)
				}
			}
		}
	}
	// the impossible visitor says why, so the player knows it was never them
	for _, b := range Bands {
		parted := Parting(HereForAnother, OutcomeParted, b)
		if !strings.Contains(parted, "met last week") {
			t.Errorf("parting must say it was never about this dog: %s", parted)
		}
	}
	// and staying with you reads differently from a polite goodbye
	warm := Parting(HereForAnother, OutcomeParted, BandWarming)
	polite := Parting(HereForAnother, OutcomeParted, BandWatching)
	if warm == polite {
		t.Error("parting well must not read the same as parting politely")
	}
	if !strings.Contains(warm, "late for him") {
		t.Errorf("the warm parting must say what the dog changed: %s", warm)
	}
	// the visit ending without a result still lands somewhere, it never
	// leaves the player on the last line of a rejection
	movedOn := Parting(QuietSeeker, OutcomeMovedOn, BandDistant)
	if !strings.Contains(movedOn, "chin back down") {
		t.Errorf("moving on must land back in the room: %s", movedOn)
	}
	// and one screen never says row three times
	screen := strings.ToLower(strings.Join(QuietSeeker.Arrival, " ") + " " +
		BodyLanguage(QuietSeeker, BandDistant) + " " + movedOn)
	if strings.Count(screen, "row") > 2 {
		t.Errorf("the word row is worn out on this screen: %s", screen)
	}
}

func TestArchetypeForIsStableAndCoversBoth(t *testing.T) {
	// the same position always finds the same person, so a resume does
	// not change who is at the gate
	if ArchetypeFor(1, 0).ID != ArchetypeFor(1, 0).ID {
		t.Error("archetype choice must be deterministic")
	}
	// the day has to matter, or every morning opens with the same face
	if ArchetypeFor(1, 0).ID == ArchetypeFor(2, 0).ID {
		t.Error("a new day must not open with the same visitor")
	}
	if ArchetypeFor(1, 0).ID != ArchetypeFor(1, 1).ID && ArchetypeFor(1, 1).ID != ArchetypeFor(2, 0).ID {
		t.Error("the order must roll forward across the day boundary")
	}
	seen := map[string]bool{}
	for day := 1; day <= 3; day++ {
		for nth := 0; nth < 2; nth++ {
			a := ArchetypeFor(day, nth)
			seen[a.ID] = true
			if _, ok := ByID(a.ID); !ok {
				t.Errorf("archetype %s is not findable by id", a.ID)
			}
		}
	}
	if len(seen) != len(Archetypes) {
		t.Errorf("a three day run must meet every archetype, met %d", len(seen))
	}
	if _, ok := ByID("nobody"); ok {
		t.Error("an unknown id must not resolve")
	}
}

func TestMeetIsPureAndReadable(t *testing.T) {
	for _, a := range Archetypes {
		for _, s := range Signals {
			r := Meet(a, s)
			if r.Body == "" || r.Parting == "" || r.Band == "" || r.Outcome == "" {
				t.Fatalf("%s answering %s left something empty: %+v", a.ID, s, r)
			}
			// the same input twice reads the same, nothing accumulates
			if again := Meet(a, s); again != r {
				t.Errorf("%s answering %s is not pure", a.ID, s)
			}
			// no number ever reaches the player
			for _, digit := range "0123456789" {
				if strings.ContainsRune(r.Body+r.Parting, digit) {
					t.Errorf("%s answering %s leaked a number: %s %s", a.ID, s, r.Body, r.Parting)
				}
			}
		}
	}
}
