package session

// The mismatch narrator: after every signal, two short lines. What the
// player meant is always clear to the player. What the visitor heard is
// the game's clarity and its heart at once, per the dog-perception skill.
// This table is the whole understanding system for the prototype. It is
// pure and it never touches a model.

type Mismatch struct {
	Meant string `json:"meant"`
	Heard string `json:"heard"`
}

// Lines never name the dog's pronoun: Narrate does not know the dog,
// and a guess would misgender a real animal.
var mismatches = map[Vocalization]Mismatch{
	PlayfulBark: {Meant: "come play", Heard: "too loud, too sudden"},
	AlertBark:   {Meant: "look, someone new", Heard: "something out there, not me"},
	Whine:       {Meant: "please stay a little longer", Heard: "not sure about me"},
	LowGrowl:    {Meant: "not yet, give me a moment", Heard: "maybe not this one"},
	Howl:        {Meant: "I am here, I am here", Heard: "a lot, all at once"},
	Silence:     {Meant: "I am waiting to see", Heard: "calm, watching me"},
}

// Narrate returns the two lines for a signal. Unknown signals narrate as
// silence, the safest reading, never as an error the player would see.
func Narrate(v Vocalization) Mismatch {
	if m, ok := mismatches[v]; ok {
		return m
	}
	return mismatches[Silence]
}
