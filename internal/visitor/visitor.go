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
	// Is is the verb that agrees with Subject. Every template that needs
	// it writes {is} rather than the word, so a they/them visitor reads
	// "they are still watching you" and not "they is".
	Is string
	// Object is the form after a preposition: went home with her. The
	// ending line needs it and nothing else does yet, but leaving it out
	// meant the best line in the game said a dog went home with the
	// staff, because "them" already means the staff everywhere in this
	// scene.
	Object string
}

var (
	she = Pronoun{"she", "her", "is", "her"}
	he  = Pronoun{"he", "his", "is", "him"}
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
	// Because is why they cannot, as the sentence the player reads.
	// Empty when they can choose. Every parting this visitor can reach
	// opens with it, so the reason is written here exactly once.
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
			"A man is already holding a leash when he stops at your kennel.",
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
		Because:   "{They} came for a dog {they} met last week.",
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

// bandFor grades a comfort value. Comfort itself never leaves this
// package at all, as a number or as a function. The thresholds are spread for a whole scene:
// four exchanges at up to two each, so every rung is reachable and no
// single answer decides the visit.
func bandFor(comfort int) Band {
	switch {
	case comfort <= -4:
		return BandDrifting
	case comfort <= -2:
		return BandDistant
	case comfort <= 1:
		return BandWatching
	case comfort <= 3:
		return BandWarming
	default:
		return BandClose
	}
}

// ExchangesPerScene is how many answers one visit is worth. A scene is
// four to eight beats in the design, this is the short end.
const ExchangesPerScene = 4

// comfort adds up how a visit has gone so far. It stays in this
// package: nothing outside ever sees the number.
func comfort(a Archetype, signals []Signal) int {
	total := 0
	for _, s := range signals {
		total += a.Prefers[s]
	}
	return total
}

// bodyTemplates are the whole readable language of comfort. One line
// per band, each a different thing a body does, no two alike. The
// placeholders are named, never positional, so rewriting a line can
// never render a format error at a player. These are reactions, and
// they are only ever used on the first answer of a visit.
var bodyTemplates = map[Band]string{
	BandDrifting: "{They} checks {their} phone.",
	BandDistant:  "{They} glances down the row at the next kennel.",
	BandWatching: "{They} keeps {their} eyes on you.",
	BandWarming:  "{They} crouches down to your level.",
	BandClose:    "{They} puts a hand flat against the gate.",
}

// settledTemplates are the same five rungs said as where the visitor
// now stands rather than as an answer to the last thing the dog did.
// From the second answer on, the narrator above names one signal while
// the band underneath holds the whole visit, and the two read as cause
// and effect unless the body says otherwise. Without the "still", a
// growl on the last exchange looks like what put a hand on the gate.
var settledTemplates = map[Band]string{
	BandDrifting: "{They} has {their} phone out again.",
	BandDistant:  "{They} {is} still looking down the row.",
	BandWatching: "{They} still has {their} eyes on you.",
	BandWarming:  "{They} {is} still crouched down at your level.",
	BandClose:    "{They} has not taken {their} hand off the gate.",
}

// Body is the loud line under the narrator: what the visitor is doing
// after the answers so far. The first answer gets a reaction to itself,
// every answer after that gets where the visit now stands, because the
// narrator above it names one signal and this has to speak for four.
func Body(a Archetype, signals []Signal) string {
	b := bandFor(comfort(a, signals))
	if len(signals) <= 1 {
		return a.render(bodyTemplates, b)
	}
	return a.render(settledTemplates, b)
}

// Moment is how one exchange read at the time, for the column of
// exchanges already past.
//
// It is the reaction wording, never the settled wording, and that is
// the whole point. "Still" and "again" are claims about the line above,
// and once the visit is stacked on screen the player can check them: a
// row saying she still has her eyes on you, directly under a row saying
// she glanced down the row, is simply false. A past exchange is a
// moment that happened, so it is written as one.
func Moment(a Archetype, signals []Signal) string {
	return a.render(bodyTemplates, bandFor(comfort(a, signals)))
}

// Settled is the visit so far as the player can still see it, oldest
// first and not including the answer that just landed.
//
// Runs of the same reading collapse to one row. A visit where the
// visitor did not move is a real thing and the game should say so, but
// it says it once: four identical sentences in a column is a render
// bug to anybody reading it, whatever it is to the engine.
func Settled(a Archetype, signals []Signal) []string {
	if len(signals) < 2 {
		return nil
	}
	var out []string
	for i := range signals[:len(signals)-1] {
		line := Moment(a, signals[:i+1])
		if len(out) > 0 && out[len(out)-1] == line {
			continue
		}
		out = append(out, line)
	}
	return out
}

// render fills one band's line, or nothing at all for a band that has
// none. Never a placeholder and never a format error.
func (a Archetype) render(table map[Band]string, b Band) string {
	tpl, ok := table[b]
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
		"{is}", a.Pronoun.Is,
		"{them}", a.Pronoun.Object,
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

// parting is the last line of the visit. Consequence wording only: it
// says what happened, never how the player did, in the same present
// tense as the arrival and the body. The band is here because parting
// well should not read the same as parting politely.
//
// A visitor who could never choose says why on every outcome, the cold
// ones included. That is the whole exculpation, and the cold ones are
// exactly where a player needs to hear it.
func parting(a Archetype, o Outcome, b Band) string {
	warm := b == BandWarming || b == BandClose
	switch o {
	case OutcomeAsked:
		return a.say("{They} stops at the desk on the way out and asks your name.")
	case OutcomeParted:
		if a.Because == "" {
			return a.say("{They} says goodbye on {their} way past.")
		}
		// someone who could never take this dog home still leaves
		// differently for having stayed, and the line says what changed
		if warm {
			return a.say(a.Because + " {They} {is} late for that one now.")
		}
		return a.say(a.Because + " {They} says goodbye on {their} way past.")
	default:
		// the visit ending is not a verdict, so the beat lands back in
		// the room rather than on the leaving
		if a.Because != "" {
			return a.say(a.Because + " You put your chin back down on the cold floor.")
		}
		return a.say("{They} carries on. You put your chin back down on the cold floor.")
	}
}

// onwardLabels are what the button says between exchanges. The visit
// has a shape and the player should feel where they are in it without
// being given a count: a scene that says "2 of 4" is a form, not an
// afternoon. Each label is different, so no two beats read the same.
// "a moment more" was a count in prose: a moment is a unit and more is
// a remainder, so at the third beat of four it resolves to "one more".
// "the visit goes on" shared its frame with the closer and spent the
// ending early, and a visit is a scene name rather than something a dog
// lives inside. What is left is two weightless connectives and then the
// game noticing she has stayed.
var onwardLabels = []string{
	"and then",
	"and again",
	"still at the gate",
}

// Onward is the label for leaving exchange nth of total, counting from
// one. The last exchange ends the visit and says so.
func Onward(nth, total int) string {
	if nth >= total {
		return "the day goes on"
	}
	if nth < 1 {
		nth = 1
	}
	if nth-1 < len(onwardLabels) {
		return onwardLabels[nth-1]
	}
	// a scene is four to eight beats in the design, so a longer visit
	// than the labels cover falls back to the one label that can repeat
	// honestly. "still at the gate" three beats running would promise
	// something, "and again" only says the visit did not stop.
	return onwardLabels[1]
}

// Ending is everything the close of a visit leaves behind.
type Ending struct {
	Outcome Outcome
	Shape   Shape
	// Body is where the visitor stands at the end of the visit
	Body string
	// Arc names the shape of the whole visit in one line
	Arc     string
	Parting string
}

// Close reads the finished visit: where it ended, how it moved, and
// what the visitor does on the way out.
func Close(a Archetype, signals []Signal) Ending {
	band := bandFor(comfort(a, signals))
	outcome := OutcomeFor(a, band)
	shape := shapeOf(a, signals)
	return Ending{
		Outcome: outcome,
		Shape:   shape,
		Body:    Body(a, signals),
		Arc:     arcLine(a, shape),
		Parting: parting(a, outcome, band),
	}
}

// Shape is how a visit moved, read back at the end. It is a shape, not
// a score: none of the four is a pass or a fail.
type Shape string

const (
	ShapeWarmed Shape = "warmed"
	ShapeCooled Shape = "cooled"
	ShapeSteady Shape = "steady"
	// ShapeTurned is a visit that moved and came back to where it began
	ShapeTurned Shape = "turned"
)

// shapeOf walks the visit one answer at a time and names the movement.
// It starts from where the visitor stood before the dog made a sound,
// not from where the first answer left them, because "than when they
// stopped" is what the arc line promises the player.
func shapeOf(a Archetype, signals []Signal) Shape {
	first := rungOf(bandFor(0))
	last, low, high := first, first, first
	total := 0
	for _, s := range signals {
		total += a.Prefers[s]
		last = rungOf(bandFor(total))
		low = min(low, last)
		high = max(high, last)
	}
	switch {
	case last > first:
		return ShapeWarmed
	case last < first:
		return ShapeCooled
	case high > first || low < first:
		return ShapeTurned
	default:
		return ShapeSteady
	}
}

// rungOf is how far up the ladder a band sits, so movement can be
// compared. The ladder order is the only thing that carries meaning.
func rungOf(b Band) int {
	for i, x := range Bands {
		if x == b {
			return i
		}
	}
	return 0
}

// arcLines name the shape in the same present tense as the rest of the
// visit. Consequence wording: what moved, never how the player did.
//
// None of them names a rung. "Drifting" and "close" are band words, and
// an arc that borrows one claims a position the visit may never have
// touched. Turned covers a visit that went up and came back as well as
// one that went down and came back, so it says neither. It also no
// longer says "once" or "in the middle": the column of exchanges is now
// on screen underneath it, so both were claims a player could count.
var arcLines = map[Shape]string{
	ShapeWarmed: "{They} {is} closer now than when {they} stopped.",
	ShapeCooled: "{They} {is} further off now than when {they} stopped.",
	ShapeSteady: "{They} {is} no closer and no further than when {they} stopped.",
	ShapeTurned: "{They} moves and comes back to where {they} started.",
}

// arcLine renders the shape for this visitor.
func arcLine(a Archetype, sh Shape) string {
	line, ok := arcLines[sh]
	if !ok {
		return ""
	}
	return a.say(line)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
