// Package visitor owns who comes to the gate and how they read the
// dog's answer. Everything here is pure: an archetype plus a signal
// gives a comfort shift, a band, a body language line and an outcome.
// No clock, no model, no network.
//
// Comfort is never a number the player sees. It is only ever readable
// as body language, and the two line mismatch narrator stays the only
// place intent and perception are named.
package visitor

import "strings"

// Signal mirrors session.Vocalization by value. The two vocabularies
// are kept in step by a test in the session package, so this file
// never has to import the game state.
type Signal string

const (
	PlayfulBark Signal = "playful_bark"
	AlertBark   Signal = "alert_bark"
	Whine       Signal = "whine"
	LowGrowl    Signal = "low_growl"
	Howl        Signal = "howl"
	Silence     Signal = "silence"
)

// Signals is every signal a visitor can be answered with. Each
// archetype must have an opinion about all of them.
var Signals = []Signal{PlayfulBark, AlertBark, Whine, LowGrowl, Howl, Silence}

// Pronoun is the visitor's own, chosen when the archetype is written.
// Visitors are invented people, so nothing here is a guess.
type Pronoun struct {
	Subject    string
	Possessive string
}

var (
	she = Pronoun{"she", "her"}
	he  = Pronoun{"he", "his"}
)

// Archetype is a kind of person who stops at a kennel. Preferences are
// hidden from the player forever, they are only ever felt.
type Archetype struct {
	ID      string
	Pronoun Pronoun
	// Arrival is what the player sees when this visitor appears
	Arrival []string
	// Prefers is the comfort shift each signal makes for this person
	Prefers map[Signal]int
	// CanChoose is false when this visitor could never take this dog
	// home, for reasons that are theirs and not the dog's. Their good
	// outcome is parting well, and it is worth playing for.
	CanChoose bool
	// Because is why they cannot, in a few words. Empty when they can
	// choose. The parting copy says it plainly in their own voice.
	Because string
}

// The two archetypes for the prototype: one who is looking, and one who
// already found someone. Neither reason is ever about this dog's worth.
var (
	QuietSeeker = Archetype{
		ID:      "quiet-seeker",
		Pronoun: she,
		Arrival: []string{
			"A woman stops at your gate with her coat still buttoned.",
			"She has walked this row twice already.",
		},
		Prefers: map[Signal]int{
			Silence:     2,
			Whine:       1,
			AlertBark:   0,
			LowGrowl:    -1,
			PlayfulBark: -1,
			Howl:        -2,
		},
		CanChoose: true,
	}

	HereForAnother = Archetype{
		ID:      "here-for-another",
		Pronoun: he,
		Arrival: []string{
			"A man stops in front of your kennel with a leash already in his hand.",
			"He looks past you more than at you.",
		},
		// quiet is what reaches someone who is already passing through.
		// the loud signals leave him where he is, they never warm him,
		// or the narrator and his body would say opposite things
		Prefers: map[Signal]int{
			Whine:       1,
			Silence:     1,
			PlayfulBark: 0,
			AlertBark:   0,
			Howl:        0,
			LowGrowl:    -1,
		},
		CanChoose: false,
		Because:   "he came for a dog he met last week",
	}
)

// Archetypes in the order visitors arrive.
var Archetypes = []Archetype{QuietSeeker, HereForAnother}

// ArchetypeFor picks who is at the gate from where the run is, so a
// resume always finds the same person waiting. The day shifts the
// order so a new day does not open with the same face. With only two
// archetypes written, people do come back over a three day run, which
// is what more archetypes will fix.
func ArchetypeFor(day, nth int) Archetype {
	i := (day - 1) + nth
	if i < 0 {
		i = 0
	}
	return Archetypes[i%len(Archetypes)]
}

// ByID finds an archetype for a recorded encounter.
func ByID(id string) (Archetype, bool) {
	for _, a := range Archetypes {
		if a.ID == id {
			return a, true
		}
	}
	return Archetype{}, false
}

// Band is how the visitor is holding themselves. Five steps, and every
// one of them reads differently at a glance, so a player who never saw
// a tutorial can tell whether it is going well.
type Band string

const (
	BandDrifting Band = "drifting"
	BandDistant  Band = "distant"
	BandWatching Band = "watching"
	BandWarming  Band = "warming"
	BandClose    Band = "close"
)

// Bands from least to most comfortable, the order the player feels.
var Bands = []Band{BandDrifting, BandDistant, BandWatching, BandWarming, BandClose}

// BandFor grades a comfort value. Comfort itself never leaves this
// package as a number.
func BandFor(comfort int) Band {
	switch {
	case comfort <= -2:
		return BandDrifting
	case comfort == -1:
		return BandDistant
	case comfort == 0:
		return BandWatching
	case comfort == 1:
		return BandWarming
	default:
		return BandClose
	}
}

// bodyTemplates are the whole readable language of comfort. One line
// per band, each a different thing a body does, no two alike. The
// placeholders are named, never positional, so rewriting a line can
// never render a format error at a player.
var bodyTemplates = map[Band]string{
	BandDrifting: "{They} checks {their} phone.",
	BandDistant:  "{They} glances down the row at the next kennel.",
	BandWatching: "{They} stays where {they} is, watching you.",
	BandWarming:  "{They} crouches down to your level.",
	BandClose:    "{They} puts a hand flat against the gate.",
}

// BodyLanguage renders what the player sees for this band.
func BodyLanguage(a Archetype, b Band) string {
	tpl, ok := bodyTemplates[b]
	if !ok {
		return ""
	}
	return a.say(tpl)
}

// say fills a line with this visitor's own pronouns.
func (a Archetype) say(line string) string {
	return strings.NewReplacer(
		"{They}", capitalize(a.Pronoun.Subject),
		"{they}", a.Pronoun.Subject,
		"{their}", a.Pronoun.Possessive,
	).Replace(line)
}

// Outcome is what the visit left behind. None of the three is a loss.
type Outcome string

const (
	// OutcomeAsked is someone going to the desk to ask about you
	OutcomeAsked Outcome = "asked"
	// OutcomeParted is a goodbye that landed, the good end of a visit
	// with someone who was never going to take you home
	OutcomeParted Outcome = "parted"
	// OutcomeMovedOn is the row carrying on, which it does all day
	OutcomeMovedOn Outcome = "moved_on"
)

// OutcomeFor reads the outcome off the band. A visitor who cannot
// choose can still part well, which is the best this visit has, and
// the game treats it as a real result.
func OutcomeFor(a Archetype, b Band) Outcome {
	warm := b == BandWarming || b == BandClose
	switch {
	case warm && a.CanChoose:
		return OutcomeAsked
	case warm || b == BandWatching:
		return OutcomeParted
	default:
		return OutcomeMovedOn
	}
}

// Parting is the last line of the visit. Consequence wording only: it
// says what happened, never how the player did, in the same present
// tense as the arrival and the body. The band is here because parting
// well should not read the same as parting politely.
func Parting(a Archetype, o Outcome, b Band) string {
	warm := b == BandWarming || b == BandClose
	switch o {
	case OutcomeAsked:
		return a.say("{They} stops at the desk on the way out and asks your name.")
	case OutcomeParted:
		if a.Because == "" {
			return a.say("{They} says goodbye through the gate.")
		}
		// someone who could never take this dog home still leaves
		// differently for having stayed, and the line says what changed
		if warm {
			return a.say("{They} came for a dog {they} met last week. {They} is late for him now.")
		}
		return a.say("{They} came for a dog {they} met last week. {They} says goodbye through the gate.")
	default:
		// the visit ending is not a verdict, so the beat lands back in
		// the room rather than on the leaving
		return a.say("{They} moves on. You put your chin back down on the cold floor.")
	}
}

// Result is everything one exchange leaves for the player to read.
type Result struct {
	Band    Band
	Body    string
	Outcome Outcome
	Parting string
}

// Meet is the whole comfort function for a single exchange: a signal
// meets a person, and what comes back is a body, not a score.
func Meet(a Archetype, s Signal) Result {
	band := BandFor(a.Prefers[s])
	outcome := OutcomeFor(a, band)
	return Result{
		Band:    band,
		Body:    BodyLanguage(a, band),
		Outcome: outcome,
		Parting: Parting(a, outcome, band),
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
