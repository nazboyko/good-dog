// Package session owns the truth of one run: which beat the player is
// on, what the client is allowed to know at that beat, and what the
// player chose. The client renders and asks to advance, it never decides.
package session

// Beat is one screen of the run. Order is fixed by rails.
type Beat string

const (
	BeatWake     Beat = "wake"
	BeatScent    Beat = "scent"
	BeatVisitor  Beat = "visitor"
	BeatNight    Beat = "night"
	BeatEpilogue Beat = "epilogue"
	BeatDone     Beat = "done"
)

// rails is the whole prototype. A single day, one of everything, then
// the reveal. Nothing branches yet.
var rails = []Beat{BeatWake, BeatScent, BeatVisitor, BeatNight, BeatEpilogue, BeatDone}

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

func ValidVocalization(v Vocalization) bool {
	for _, known := range vocalizations {
		if known == v {
			return true
		}
	}
	return false
}

// next returns the beat after b, or done when the run is over.
func next(b Beat) Beat {
	for i, r := range rails {
		if r == b && i+1 < len(rails) {
			return rails[i+1]
		}
	}
	return BeatDone
}
