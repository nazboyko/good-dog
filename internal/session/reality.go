package session

import (
	"fmt"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/radio"
)

// The one sentence the reveal says about the real dog.
//
// Two different clocks meet here and only one of them is real. The
// ending is what happened in your three days, which the game made up.
// The status is what the listing says right now, which it did not. When
// they disagree the listing wins, always: a dog who was adopted on
// Saturday cannot be told to a player as still waiting because the
// fiction happened to end that way.
//
// Every branch is a fact about a web page, in the same grammar: still
// listed, no longer listed, was adopted. The word adopted appears in
// exactly one of them, and only because the listing said it first.

// RealityLine is what is true about this dog now, in one sentence.
func RealityLine(dog animal.Animal, org animal.Organization, ending Ending) string {
	name := dog.Name
	where := fmt.Sprintf("%s in %s, %s", radio.OrgShort(org.Name), org.City, StateName(org.State))

	switch dog.Status {
	case animal.StatusAdoptedConfirmed:
		// the listing's own claim, in the listing's own words, with the
		// date it gave
		if dog.AdoptedOn != "" {
			return fmt.Sprintf("%s is real, and was adopted on %s.", name, dog.AdoptedOn)
		}
		return fmt.Sprintf("%s is real, and has been adopted.", name)

	case animal.StatusActive:
		// still listed. Waiting is only allowed when the game has not
		// just said somebody took her home
		if StillWaiting(ending) {
			return fmt.Sprintf("%s is real, and waiting with %s.", name, where)
		}
		return fmt.Sprintf("%s is real, and still listed with %s.", name, where)

	default:
		// The page stopped saying anything and did not say why, or the
		// status is one this code has never heard of. Either way that is
		// all the game knows, so that is all it says: not left, which is
		// a claim about the dog, and not any guess at the reason, since
		// some of the reasons are not happy ones. An unknown status fails
		// closed to this line rather than open to waiting: not knowing is
		// not the same as knowing she is there.
		return fmt.Sprintf("%s is real, and no longer listed with %s.", name, radio.OrgShort(org.Name))
	}
}

// Seam reports whether the reveal shows "That was your three days."
// before the truth line. It marks the place where the fiction and the
// listing part ways. For a listed dog that only happens after a chosen
// ending, where the game just said she went home and now has to say she
// is listed. For an adopted dog it happens on every ending: the player
// spent three days in a kennel with a dog who was not in it.
func Seam(dog animal.Animal, ending Ending) bool {
	if dog.Status == animal.StatusAdoptedConfirmed {
		return true
	}
	return !StillWaiting(ending)
}
