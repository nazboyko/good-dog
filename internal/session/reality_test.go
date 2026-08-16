package session

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
)

func ahs() animal.Organization {
	return animal.Organization{
		Name:  "Animal Humane Society, Golden Valley",
		City:  "Golden Valley",
		State: "MN",
	}
}

func dogNamed(name string, status animal.Status, adoptedOn string) animal.Animal {
	return animal.Animal{Name: name, Status: status, AdoptedOn: adoptedOn}
}

// A dog whose listing says she was adopted is never told to a player as
// waiting, whichever way the three days happened to end. The fiction is
// the game's, the listing is not, and the listing wins.
func TestAnAdoptedDogIsNeverCalledWaiting(t *testing.T) {
	bella := dogNamed("Bella", animal.StatusAdoptedConfirmed, "August 15, 2026")
	for _, ending := range []Ending{EndingChosen, EndingAnotherDog, EndingNobodyToday} {
		line := RealityLine(bella, ahs(), ending)
		if strings.Contains(line, "waiting") || strings.Contains(line, "still listed") {
			t.Errorf("ending %s: %q says she is still there", ending, line)
		}
		if !strings.Contains(line, "August 15, 2026") {
			t.Errorf("ending %s: %q drops the date her listing gives", ending, line)
		}
		if !strings.Contains(line, "adopted") {
			t.Errorf("ending %s: %q never says the true thing", ending, line)
		}
	}
}

// The word adopted is the listing's claim, so it may only appear where
// the listing made it. Anywhere else it is the game inventing an outcome
// for a real animal.
func TestOnlyAConfirmedAdoptionMaySayAdopted(t *testing.T) {
	others := []animal.Status{
		animal.StatusActive,
		animal.StatusRemovedUnknown,
		animal.StatusTransferred,
		animal.StatusUnavailable,
	}
	for _, st := range others {
		for _, ending := range []Ending{EndingChosen, EndingAnotherDog, EndingNobodyToday} {
			line := RealityLine(dogNamed("Keno", st, ""), ahs(), ending)
			if strings.Contains(strings.ToLower(line), "adopt") {
				t.Errorf("status %s, ending %s: %q claims an adoption nobody confirmed", st, ending, line)
			}
			if strings.Contains(line, "went home") {
				t.Errorf("status %s: %q reads as a homecoming", st, line)
			}
		}
	}
}

// A listing that stopped saying anything did not say why, and some of
// the reasons are not happy ones. All the game knows is that the page
// stopped saying anything, so that is all it says.
func TestAGoneDogGetsCarefulWording(t *testing.T) {
	for _, st := range []animal.Status{animal.StatusRemovedUnknown, animal.StatusTransferred, animal.StatusUnavailable} {
		line := RealityLine(dogNamed("Lori", st, ""), ahs(), EndingNobodyToday)
		if !strings.Contains(line, "no longer listed") {
			t.Errorf("status %s: %q, want the careful wording", st, line)
		}
		// left is a claim about the dog, and the game only has a claim
		// about a page
		for _, guess := range []string{"adopt", "home", "family", "died", "put down", "left"} {
			if strings.Contains(strings.ToLower(line), guess) {
				t.Errorf("status %s: %q guesses at why with %q", st, line, guess)
			}
		}
	}
}

// The dog still on the listing keeps the line the game was built around,
// and the ending still decides between waiting and still listed.
func TestAListedDogStillTurnsOnTheEnding(t *testing.T) {
	keno := dogNamed("Keno", animal.StatusActive, "")
	if got := RealityLine(keno, ahs(), EndingNobodyToday); !strings.Contains(got, "waiting with Animal Humane Society in Golden Valley, Minnesota") {
		t.Errorf("nobody came, so she is waiting: %q", got)
	}
	// the game just said somebody took her home, so the reveal cannot
	// also say she is waiting
	if got := RealityLine(keno, ahs(), EndingChosen); strings.Contains(got, "waiting") {
		t.Errorf("chosen ending still says waiting: %q", got)
	} else if !strings.Contains(got, "still listed") {
		t.Errorf("chosen ending lost the listing: %q", got)
	}
}

// The seam marks where the fiction and the listing part ways. For a
// listed dog that is only after a chosen ending. For an adopted dog it
// is every ending: the player spent three days in a kennel with a dog
// who was not in it.
func TestTheSeamShowsOnEveryAdoptedEnding(t *testing.T) {
	bella := dogNamed("Bella", animal.StatusAdoptedConfirmed, "August 15, 2026")
	keno := dogNamed("Keno", animal.StatusActive, "")
	for _, ending := range []Ending{EndingChosen, EndingAnotherDog, EndingNobodyToday} {
		if !Seam(bella, ending) {
			t.Errorf("adopted, ending %s: no seam", ending)
		}
	}
	if !Seam(keno, EndingChosen) {
		t.Error("listed, chosen: the seam is the whole point of the chosen path")
	}
	if Seam(keno, EndingNobodyToday) || Seam(keno, EndingAnotherDog) {
		t.Error("listed, not chosen: the ending and the listing agree, no seam")
	}
}

// The night's own story is the other place the game says the dog is
// real, and it must not say she is still here when her page says she
// went home, or that she has been waiting.
func TestTheOwnStoryFollowsTheListing(t *testing.T) {
	sheet := &dogsheet.DogSheet{RadioSeed: dogsheet.NarrativeInference{Value: "Meet Bella."}}
	bella := dogNamed("Bella", animal.StatusAdoptedConfirmed, "August 15, 2026")
	bella.Sex = "Female"
	bella.LongStay = true
	joined := strings.Join(RadioStory(bella, sheet), " ")
	if strings.Contains(joined, "still here") || strings.Contains(joined, "waiting") {
		t.Errorf("an adopted dog's own night says she is still here: %q", joined)
	}
	if !strings.Contains(joined, "went home") {
		t.Errorf("an adopted dog's own night never says the true thing: %q", joined)
	}

	keno := dogNamed("Keno", animal.StatusActive, "")
	keno.Sex = "Male"
	if got := strings.Join(RadioStory(keno, sheet), " "); !strings.Contains(got, "still here") {
		t.Errorf("a listed dog's night lost its ending: %q", got)
	}

	gone := dogNamed("Lori", animal.StatusRemovedUnknown, "")
	got := strings.Join(RadioStory(gone, sheet), " ")
	for _, claim := range []string{"still here", "went home", "waiting", "adopt"} {
		if strings.Contains(strings.ToLower(got), claim) {
			t.Errorf("a gone dog's night claims %q: %q", claim, got)
		}
	}
	if !strings.Contains(got, "real") {
		t.Errorf("a gone dog is still real: %q", got)
	}
}

// Ground truth is the fixture file, not this package's idea of it. If a
// dog is marked adopted, the date the listing gave has to travel with
// the status, because the reveal says it out loud.
func TestEveryAdoptedFixtureCarriesItsDate(t *testing.T) {
	raw, err := os.ReadFile("../../fixtures/dogs.json")
	if err != nil {
		t.Skip("no fixtures here")
	}
	var file struct {
		Animals []animal.Animal `json:"animals"`
		Dogs    []animal.Animal `json:"dogs"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	dogs := file.Animals
	if len(dogs) == 0 {
		dogs = file.Dogs
	}
	if len(dogs) == 0 {
		t.Fatal("no dogs read from the fixture file")
	}
	for _, d := range dogs {
		if d.Synthetic {
			continue
		}
		switch d.Status {
		case animal.StatusAdoptedConfirmed:
			if d.AdoptedOn == "" {
				t.Errorf("%s (%s) is adopted with no date, so the reveal cannot say when", d.Name, d.ID)
			}
		default:
			if d.AdoptedOn != "" {
				t.Errorf("%s (%s) carries an adoption date but is %s", d.Name, d.ID, d.Status)
			}
		}
	}
}
