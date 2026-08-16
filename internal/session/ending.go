package session

// How a run ends, and the one place the game's story meets the world's
// facts.
//
// The seam is this. The three days are fiction: a real dog's listing
// was the raw material, but nobody was chosen, nothing happened, and
// the player made it happen by playing. The listing is not fiction. She
// is on it right now, whatever the game just said.
//
// So the ending line is what happened in your three days, and the
// reveal that follows is what is true today, and neither is allowed to
// pretend to be the other. The chosen ending is where that is sharpest:
// the game has just said she went home, and the next screen has to show
// a dog who is still listed without calling the ending a lie and
// without quietly dropping it either.
//
// The reveal does not say "waiting" after a chosen ending. It says
// still listed, which is a fact about a web page rather than a claim
// about a dog, and set against "she went home with them today" it says
// the thing the game is actually for: that ending was yours to give her
// and it has not happened yet.

// EndingLine is the quiet line before the reveal staging begins. Short,
// past tense, and it never reaches for the player's feelings: the
// photograph two beats later does that on its own.
func EndingLine(e Ending, name, withThem string) string {
	switch e {
	case EndingChosen:
		// "with them" would read as the staff: they is what the shelter
		// workers are called everywhere else in this scene
		return name + " went home with " + withThem + " today."
	case EndingAnotherDog:
		// she is the woman the player just spent six exchanges with, not
		// a stranger. The line says nothing about this dog on purpose.
		//
		// The dog in the next kennel is deliberately nobody: no name, no
		// photo, no status. The day neighbour kennels hold real dogs from
		// the pool, this line becomes an invented adoption status for a
		// real animal, which is the one thing layer one forbids. Pinned
		// by a test.
		return "She left with the dog in the next kennel."
	case EndingNobodyToday:
		return "Nobody came for " + name + " today."
	default:
		return ""
	}
}

// StillWaiting reports whether the reveal may say this dog is waiting.
//
// After any ending but chosen she is waiting, and saying so is both
// true and the point. After chosen the game has just put her in
// somebody's car, so the reveal states the fact it can defend, which is
// that the listing is still open, and lets the player feel the distance
// between the two without being told what to feel about it.
func StillWaiting(e Ending) bool { return e != EndingChosen }
