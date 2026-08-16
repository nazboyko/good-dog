// Package session owns the truth of one run: which beat the player is
// on, what the client is allowed to know at that beat, and what the
// player chose. The client renders and asks to advance, it never decides.
package session

// Beat is one screen of the run. Order is fixed by rails.
type Beat string

const (
	BeatWake    Beat = "wake"
	BeatScent   Beat = "scent"
	BeatVisitor Beat = "visitor"
	// BeatAdoption is the last visitor of the run, in the meeting room.
	// Same machinery as a visit, longer, and its close decides how the
	// three days ended.
	BeatAdoption Beat = "adoption"
	BeatNight    Beat = "night"
	BeatEpilogue Beat = "epilogue"
	BeatDone     Beat = "done"
)

// Vocalization is the six option panel from the bark-input skill.
// Silence is a deliberate choice, not the absence of one.
type Vocalization string

const (
	PlayfulBark Vocalization = "playful_bark"
	AlertBark   Vocalization = "alert_bark"
	Whine       Vocalization = "whine"
	LowGrowl    Vocalization = "low_growl"
	Howl        Vocalization = "howl"
	Silence     Vocalization = "silence"
)

var vocalizations = []Vocalization{PlayfulBark, AlertBark, Whine, LowGrowl, Howl, Silence}

// Vocalizations is every sound the player can choose. Exported so the
// packages that have to cover all of them, like the recordings, can
// check against this list rather than against a copy of it that drifts.
func Vocalizations() []Vocalization {
	out := make([]Vocalization, len(vocalizations))
	copy(out, vocalizations)
	return out
}

func ValidVocalization(v Vocalization) bool {
	for _, known := range vocalizations {
		if known == v {
			return true
		}
	}
	return false
}
