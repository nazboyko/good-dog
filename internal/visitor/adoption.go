package visitor

// Adoption day. The last person to stand at the gate, except there is
// no gate: a meeting room, a floor, and a door somebody will leave
// through. The scene the whole run has been building toward.
//
// Nothing new is invented here. It is the same comfort function, the
// same five rungs, the same mismatch narrator. What changes is that the
// scene is longer, the room has its own furniture, and where the
// visitor starts depends on the three days behind it.
//
// The room matters more than it sounds. The first version of this scene
// reused the row's body lines, so the warmest rung of the most
// important beat in the game read "she puts a hand flat against the
// gate" in a room with no gate, and the coldest read "glances down the
// row at the next kennel" in a room with no row. A shared vocabulary is
// only shared until the furniture changes.

// AdoptionExchanges is how many answers the last scene is worth. Longer
// than a visit, because it is the one that decides the run, and short
// enough that it is still a scene rather than a level.
const AdoptionExchanges = 6

// Past is what the run has already been: one entry per visit that
// happened, in the order they happened.
type Past struct {
	Outcome Outcome
}

// Adopter is who comes on the last day. There is exactly one, and it is
// built rather than picked, because by day three the game knows enough
// about the run to say who would be standing there.
func Adopter(history []Past) Archetype {
	opening := "A woman is waiting in the meeting room when they walk you in."
	if warmth(history) > 0 {
		// somebody at the desk has a note on file. It is not a reward
		// for playing well, it is that she has read it
		opening = "A woman is waiting in the meeting room. Somebody at the desk has told her about you."
	}
	return Archetype{
		ID:      "adopter",
		Pronoun: she,
		Arrival: []string{
			opening,
			// this line is always here: it is the image the scene is
			// built on, and the labels below promise she is still sitting
			"She sits down on the floor without being asked to.",
		},
		Prefers: map[Signal]int{
			Silence:     2,
			Whine:       1,
			AlertBark:   0,
			PlayfulBark: 0,
			LowGrowl:    -1,
			Howl:        -1,
		},
		CanChoose: true,
	}
}

// meetingRoom is the reaction vocabulary for the last scene. Same five
// rungs, furniture that is actually in the room: a floor, a door, and
// the space between two people sitting on it.
var meetingRoom = map[Band]string{
	BandDrifting: "{They} checks {their} phone.",
	BandDistant:  "{They} looks over at the door.",
	BandWatching: "{They} keeps {their} eyes on you.",
	BandWarming:  "{They} leans in toward you.",
	BandClose:    "{They} puts a hand flat on the floor between you.",
}

// meetingRoomSettled is the same five rungs said as where she now is,
// for every answer after the first.
var meetingRoomSettled = map[Band]string{
	BandDrifting: "{They} has {their} phone out again.",
	BandDistant:  "{They} {is} still looking at the door.",
	BandWatching: "{They} still has {their} eyes on you.",
	BandWarming:  "{They} {is} still leaning in.",
	BandClose:    "{They} has not moved {their} hand off the floor.",
}

// headStart is where the last visitor begins, before the dog has done
// anything, and it is capped at one rung.
//
// Two is too much. The meeting room is six exchanges on a ladder spread
// for four, so a warm run plus a head start of two reached the top rung
// on the first answer and the scene had nowhere left to go for five
// more. A run that went well opens the door slightly, it does not walk
// through it for the player.
func headStart(history []Past) int {
	if warmth(history) > 0 {
		return 1
	}
	return 0
}

// warmth counts how the three days went, in the only terms the game
// records: someone asked about this dog, or someone parted well.
func warmth(history []Past) int {
	n := 0
	for _, p := range history {
		switch p.Outcome {
		case OutcomeAsked:
			n += 2
		case OutcomeParted:
			n++
		}
	}
	if n >= 3 {
		return 1
	}
	return 0
}

// adoptionBand grades the last scene on its own ladder.
//
// bandFor is spread for four exchanges at up to two each. This scene is
// six, so the same comfort arrives sooner and the top rung would be
// reached and held for most of the scene. The thresholds are stretched
// to match the length, which keeps every rung reachable and keeps the
// last answer able to change the ending.
func adoptionBand(a Archetype, history []Past, signals []Signal) Band {
	total := comfort(a, signals) + headStart(history)
	// six answers at up to two each is a range of twelve, plus a rung of
	// head start. Spread to match: the top arrives near the end of the
	// scene rather than a third of the way in, and the last answer can
	// still move it.
	switch {
	case total <= -7:
		return BandDrifting
	case total <= -3:
		return BandDistant
	case total <= 3:
		return BandWatching
	case total <= 10:
		return BandWarming
	default:
		return BandClose
	}
}

// CloseAdoption reads the last scene.
func CloseAdoption(a Archetype, history []Past, signals []Signal) Ending {
	band := adoptionBand(a, history, signals)
	outcome := OutcomeFor(a, band)
	shape := adoptionShape(a, history, signals)
	return Ending{
		Outcome: outcome,
		Shape:   shape,
		Body:    AdoptionBody(a, history, signals),
		Arc:     arcLine(a, shape),
		Parting: adoptionParting(a, outcome),
	}
}

// adoptionShape walks the last scene on its own ladder, so the arc line
// describes the column the player can see.
func adoptionShape(a Archetype, history []Past, signals []Signal) Shape {
	first := rungOf(adoptionBand(a, history, nil))
	last, low, high := first, first, first
	for i := range signals {
		last = rungOf(adoptionBand(a, history, signals[:i+1]))
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

// AdoptionBody is the loud line during the last scene.
func AdoptionBody(a Archetype, history []Past, signals []Signal) string {
	b := adoptionBand(a, history, signals)
	if len(signals) <= 1 {
		return a.render(meetingRoom, b)
	}
	return a.render(meetingRoomSettled, b)
}

// AdoptionSettled is the column of exchanges already past, in reaction
// wording, with runs of the same reading collapsed to one row.
func AdoptionSettled(a Archetype, history []Past, signals []Signal) []string {
	if len(signals) < 2 {
		return nil
	}
	var out []string
	for i := range signals[:len(signals)-1] {
		line := a.render(meetingRoom, adoptionBand(a, history, signals[:i+1]))
		if len(out) > 0 && out[len(out)-1] == line {
			continue
		}
		out = append(out, line)
	}
	return out
}

// adoptionParting is how the last visit ends. None of the three is a
// loss and none is a verdict on the player. The one where nobody takes
// you home lands back in the room rather than on the person leaving,
// which is the rule the row already keeps for the same outcome.
func adoptionParting(a Archetype, o Outcome) string {
	switch o {
	case OutcomeAsked:
		return a.say("{They} asks how soon {they} can take you home.")
	case OutcomeParted:
		return a.say("{They} sits with you a while longer, then goes to talk to the desk.")
	default:
		return a.say("{They} stands up and thanks them for the time. Someone clips the lead back on.")
	}
}

// adoptionLabels is the way forward inside the last scene.
//
// Every one of these has to be true in every band, because the sequence
// does not know how the scene is going. An earlier version said "she
// has not looked at the door" at a fixed exchange, which is flatly
// contradicted by the distant body line on the same screen, and "one
// more", which is the counter this game does not use.
var adoptionLabels = []string{
	"and then",
	"and again",
	"she is still sitting there",
	"the room is quiet",
	"she stays",
}

// AdoptionOnward is the button label for leaving exchange nth of total.
func AdoptionOnward(nth, total int) string {
	if nth >= total {
		return "the day goes on"
	}
	if nth < 1 {
		nth = 1
	}
	return adoptionLabels[nth-1]
}
