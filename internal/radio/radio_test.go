package radio

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
)

func neighbour(id, name, orgID, orgName, city, seed, quirk string) Neighbour {
	return Neighbour{
		Dog: animal.Animal{ID: id, Name: name, OrgID: orgID},
		Org: animal.Organization{ID: orgID, Name: orgName, City: city},
		Sheet: &dogsheet.DogSheet{
			RadioSeed: dogsheet.NarrativeInference{Value: seed},
			Quirks:    []dogsheet.NarrativeInference{{Value: quirk}},
		},
	}
}

func testPool() []Neighbour {
	return []Neighbour{
		neighbour("d1", "Keno", "ahs", "Animal Humane Society, Golden Valley", "Golden Valley",
			"Here is Keno, who watches the door more than the room.", "Sleeps with one paw over his nose."),
		neighbour("d2", "Sugar Bear", "sos", "Secondhand Hounds", "Eden Prairie",
			"Sugar Bear has opinions about the vacuum.", "Carries a blanket from one end of the run to the other."),
		neighbour("d3", "Pepper", "rsr", "Ruff Start Rescue", "Princeton",
			"Pepper knows the sound of the treat drawer.", "Leans on shins until somebody sits down."),
	}
}

// The player's own dog is the exception and has to stay one: it is the
// only story that may reach for the reveal, and only because the player
// has been living in it all day.
func TestOnlyTheOwnStoryMayNameTheReveal(t *testing.T) {
	own := []string{"Here is Venus.", "She is real, and she is still here."}
	cues := Broadcast(testPool(), own, Ranger)
	var ownCues, otherCues int
	for _, c := range cues {
		if strings.Contains(c.Line, "is real") {
			ownCues++
			if !contains(own, c.Line) {
				t.Errorf("a line outside the player's own story says real: %s", c.Line)
			}
			continue
		}
		otherCues++
	}
	if ownCues != 1 {
		t.Errorf("the reveal language appears %d times, want once", ownCues)
	}
	if otherCues < 10 {
		t.Errorf("the rest of the night is thin: %d cues", otherCues)
	}
	// and the player's dog is last, after every neighbour
	last := cues[len(cues)-1]
	if last.Speaker != SpeakerRanger {
		t.Errorf("the host closes the night, got %s", last.Speaker)
	}
	ownAt := -1
	for i, c := range cues {
		if contains(own, c.Line) && ownAt < 0 {
			ownAt = i
		}
	}
	for i, c := range cues {
		if c.Speaker == SpeakerStory && !contains(own, c.Line) && i > ownAt {
			t.Errorf("a neighbour story plays after the player's own: %s", c.Line)
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// The reveal is one beat and it belongs to the dog the player has been.
// A neighbour story that says a neighbour is real, or still waiting,
// spends that beat early and on the wrong animal.
func TestNeighbourStoriesNeverReachForTheReveal(t *testing.T) {
	cues := Broadcast(testPool(), nil, Ranger)
	forbidden := []string{
		"is real", "are real", "still here", "still waiting", "waiting for you",
		"adopt", "forever home", "take him home", "take her home", "could be yours",
	}
	for _, c := range cues {
		lower := strings.ToLower(c.Line)
		for _, phrase := range forbidden {
			if strings.Contains(lower, phrase) {
				t.Errorf("%s cue reaches for the reveal with %q: %s", c.Speaker, phrase, c.Line)
			}
		}
		if strings.Contains(c.Line, "—") || strings.Contains(c.Line, "!") {
			t.Errorf("not the house voice: %s", c.Line)
		}
	}
}

// Every story ends on the dog's own name and the real place they are,
// and then it stops.
func TestEveryStoryEndsOnARealNameAndPlace(t *testing.T) {
	pool := testPool()
	cues := Broadcast(pool, nil, Ranger)
	for _, n := range pool {
		// the naming line is the host's now: a dog announcing its own
		// name and shelter in the third person is a station ident
		var last string
		for _, c := range cues {
			if strings.Contains(c.Line, n.Dog.Name) {
				last = c.Line
				if c.Speaker != SpeakerRanger {
					t.Errorf("%s names themselves, that is the host's line: %s", n.Dog.Name, c.Line)
				}
			}
		}
		if last == "" {
			t.Fatalf("%s never got a story", n.Dog.Name)
		}
		if !strings.HasPrefix(last, "That is "+n.Dog.Name) {
			t.Errorf("%s story does not end on the naming line: %s", n.Dog.Name, last)
		}
		if !strings.Contains(last, OrgShort(n.Org.Name)) || !strings.Contains(last, n.Org.City) {
			t.Errorf("%s naming line must carry the real place: %s", n.Dog.Name, last)
		}
		// the org's legal tail does not belong in a spoken line
		if strings.Contains(last, ",") && strings.Contains(last, "Golden Valley,") {
			t.Errorf("%s naming line kept the org's full legal name: %s", n.Dog.Name, last)
		}
	}
}

func TestCuesOnlyEverMoveForward(t *testing.T) {
	cues := Broadcast(testPool(), []string{"Here is Venus.", "She is real, and she is still here."}, Ranger)
	if len(cues) < 8 {
		t.Fatalf("a night with three dogs is more than %d cues", len(cues))
	}
	last := -1
	for i, c := range cues {
		if int(c.At) <= last {
			t.Errorf("cue %d lands at or before the one before it: %v", i, c.At)
		}
		last = int(c.At)
		if strings.TrimSpace(c.Line) == "" {
			t.Errorf("cue %d is empty", i)
		}
	}
	// the host opens and closes
	if cues[0].Speaker != SpeakerRanger || cues[len(cues)-1].Speaker != SpeakerRanger {
		t.Error("the host opens and closes the night")
	}
}

// A dog at the player's own shelter would name that shelter, and the
// shelter is part of a reveal the player has not reached.
func TestNeighboursLeaveOutThePlayersOwnShelter(t *testing.T) {
	pool := testPool()
	playing := animal.Animal{ID: "venus", OrgID: "rsr"}
	picked := Neighbours(pool, playing, "rsr")
	for _, n := range picked {
		if n.Org.ID == "rsr" {
			t.Errorf("%s is at the player's own shelter and would name it", n.Dog.Name)
		}
	}
	if len(picked) != 2 {
		t.Errorf("two of the three are elsewhere, got %d", len(picked))
	}
	// and the dog being played is never their own neighbour
	self := append(pool, neighbour("venus", "Venus", "rsr", "Ruff Start Rescue", "Princeton", "seed", "quirk"))
	for _, n := range Neighbours(self, playing, "rsr") {
		if n.Dog.ID == "venus" {
			t.Error("the player's dog cannot be on the radio as a neighbour")
		}
	}
}

// A canned sheet has nothing true to say, so it does not get a slot.
func TestDefaultSheetsAreNotBroadcast(t *testing.T) {
	pool := testPool()
	pool[0].Sheet.Default = true
	picked := Neighbours(pool, animal.Animal{ID: "venus"}, "none")
	for _, n := range picked {
		if n.Dog.ID == "d1" {
			t.Error("a default sheet must never reach the radio")
		}
	}
	nilSheet := []Neighbour{{Dog: animal.Animal{ID: "d9", Name: "Nobody"}, Org: animal.Organization{ID: "x"}}}
	if got := Neighbours(nilSheet, animal.Animal{ID: "venus"}, "none"); len(got) != 0 {
		t.Errorf("a dog with no sheet must not be broadcast, got %d", len(got))
	}
}

func TestBroadcastIsPure(t *testing.T) {
	pool := testPool()
	a := Broadcast(pool, nil, Ranger)
	b := Broadcast(pool, nil, Ranger)
	if len(a) != len(b) {
		t.Fatalf("two runs gave %d and %d cues", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("cue %d differs between runs: %+v %+v", i, a[i], b[i])
		}
	}
	// more dogs than slots still fills exactly the slots
	big := append(testPool(), neighbour("d4", "Bella", "x", "Xavier Rescue", "Duluth", "seed", "quirk"))
	if got := Neighbours(big, animal.Animal{ID: "venus"}, "none"); len(got) != Stories {
		t.Errorf("a full pool fills %d slots, got %d", Stories, len(got))
	}
}

// The field is called at_ms and a browser sets a timer from it. A
// time.Duration marshals as nanoseconds, so without an explicit
// marshaller the fallback timer is set thirteen days out and the whole
// no-stream path silently does nothing. The unit test that was supposed
// to cover the fallback used hand written small numbers and could not
// see this, so the check has to be against the real serialization.
func TestCuesSerializeTheirOffsetInMilliseconds(t *testing.T) {
	cues := Broadcast(testPool(), nil, Ranger)
	raw, err := json.Marshal(cues)
	if err != nil {
		t.Fatal(err)
	}
	var back []struct {
		At      int64  `json:"at_ms"`
		Speaker string `json:"speaker"`
		Line    string `json:"line"`
	}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != len(cues) {
		t.Fatalf("round trip lost cues: %d of %d", len(back), len(cues))
	}
	for i, c := range back {
		want := cues[i].At.Milliseconds()
		if c.At != want {
			t.Errorf("cue %d says %d, want %d milliseconds", i, c.At, want)
		}
		// a whole night is minutes, not weeks: anything past ten minutes
		// means the units slipped again
		if c.At > 10*60*1000 {
			t.Errorf("cue %d is %d ms into the night, the units are wrong", i, c.At)
		}
	}
}

// Two dogs with the same name in one broadcast reads as a bug even
// though shelters really are full of Bellas.
func TestOneNightNeverNamesTheSameDogTwice(t *testing.T) {
	pool := []Neighbour{
		neighbour("d1", "Bella", "ahs", "Animal Humane Society", "Golden Valley", "seed one", "quirk one"),
		neighbour("d2", "Bella", "ahs", "Animal Humane Society", "Golden Valley", "seed two", "quirk two"),
		neighbour("d3", "Preston", "ahs", "Animal Humane Society", "Golden Valley", "seed three", "quirk three"),
	}
	picked := Neighbours(pool, animal.Animal{ID: "venus"}, "none")
	seen := map[string]bool{}
	for _, n := range picked {
		if seen[n.Dog.Name] {
			t.Errorf("%s is on the radio twice in one night", n.Dog.Name)
		}
		seen[n.Dog.Name] = true
	}
	if len(picked) != 2 {
		t.Errorf("two distinct names out of three dogs, got %d", len(picked))
	}
}

// The host cannot claim every dog is down the player's own row: the
// neighbours are chosen from other shelters precisely so their naming
// line does not give away the player's, and the frame has to agree.
func TestTheHostNeverClaimsOneBuilding(t *testing.T) {
	for _, c := range Broadcast(testPool(), nil, Ranger) {
		if c.Speaker != SpeakerRanger {
			continue
		}
		for _, claim := range []string{"the row", "this row", "down the hall", "next door", "in here"} {
			if strings.Contains(strings.ToLower(c.Line), claim) {
				t.Errorf("the host puts dogs from other shelters in one building: %s", c.Line)
			}
		}
	}
}

// The player's own dog is announced once. The session writes its own
// opener and the seed names her, so a third announcement from here
// pushed the naming beat to fourth in a queue and it landed flat.
func TestTheOwnSegmentIsAnnouncedOnce(t *testing.T) {
	own := []string{"This one is for the dog in the third kennel down.", "Her name is Venus."}
	cues := Broadcast(testPool(), own, Ranger)
	// everything about a neighbour, the dog's line and the host naming
	// them, counts as the neighbour segment
	lastNeighbour, firstOwn := -1, -1
	for i, c := range cues {
		if !contains(own, c.Line) && c.Line != HostOpen && c.Line != HostWhoIsUp && c.Line != HostClose {
			lastNeighbour = i
		}
		if contains(own, c.Line) && firstOwn < 0 {
			firstOwn = i
		}
	}
	if lastNeighbour < 0 || firstOwn < 0 {
		t.Fatalf("setup: neighbours at %d, own at %d", lastNeighbour, firstOwn)
	}
	for i := lastNeighbour + 1; i < firstOwn; i++ {
		t.Errorf("the host announces the own segment again at %d, the session already does it: %s",
			i, cues[i].Line)
	}
}

// Every line a player hears is a finished sentence, including the ones
// the model returned as a fragment.
func TestEveryLineIsAFinishedSentence(t *testing.T) {
	pool := testPool()
	pool[0].Sheet.Quirks[0].Value = "Sleeps with one paw over his nose"
	for _, c := range Broadcast(pool, nil, Ranger) {
		if !strings.HasSuffix(c.Line, ".") && !strings.HasSuffix(c.Line, "?") {
			t.Errorf("%s cue is a fragment: %q", c.Speaker, c.Line)
		}
	}
}

// The seed is written to the shape of a listing and carries the loosest
// claims in the sheet. It does not belong in a neighbour's story.
func TestNeighbourStoriesDoNotReadTheListingBlurb(t *testing.T) {
	pool := testPool()
	pool[0].Sheet.RadioSeed.Value = "Meet Keno, a resilient three-legged hound mix whose boundless energy shines."
	for _, c := range Broadcast(pool, nil, Ranger) {
		if strings.Contains(c.Line, "Meet ") || strings.Contains(c.Line, "resilient") {
			t.Errorf("a listing blurb reached the radio: %s", c.Line)
		}
	}
}

// Every story ends by naming a place. Three dogs from one shelter is
// that sentence three times, and by the third nobody hears it.
func TestTheNightSpreadsAcrossSheltersWhenItCan(t *testing.T) {
	pool := []Neighbour{
		neighbour("d1", "Bella", "ahs", "Animal Humane Society", "Golden Valley", "s", "q"),
		neighbour("d2", "Preston", "ahs", "Animal Humane Society", "Golden Valley", "s", "q"),
		neighbour("d3", "Arya", "ahs", "Animal Humane Society", "Golden Valley", "s", "q"),
		neighbour("d4", "Moose", "shh", "Secondhand Hounds", "Eden Prairie", "s", "q"),
		neighbour("d5", "Keno", "rr", "Rescue Roundup", "Duluth", "s", "q"),
	}
	picked := Neighbours(pool, animal.Animal{ID: "venus"}, "none")
	orgs := map[string]bool{}
	for _, n := range picked {
		orgs[n.Org.ID] = true
	}
	if len(picked) != Stories {
		t.Fatalf("wanted %d dogs, got %d", Stories, len(picked))
	}
	if len(orgs) != Stories {
		t.Errorf("three shelters were available, the night used %d: %v", len(orgs), orgs)
	}
	// but a pool with only one shelter still fills the night
	oneOrg := pool[:3]
	if got := Neighbours(oneOrg, animal.Animal{ID: "venus"}, "none"); len(got) != Stories {
		t.Errorf("a single shelter pool must still fill %d slots, got %d", Stories, len(got))
	}
}

// The night has to feel heard rather than read, so the client rolls a
// window over the neighbours' lines. That only works if the broadcast
// says which lines are the player's own: those stay, because the night
// ends on them.
func TestTheOwnSegmentIsMarkedSoItCanStay(t *testing.T) {
	own := []string{"Her name is Venus.", "She is real, and she is still here."}
	cues := Broadcast(testPool(), own, Ranger)

	var ownCues, neighbourCues int
	for _, c := range cues {
		switch c.Speaker {
		case SpeakerOwn:
			ownCues++
			if !contains(own, c.Line) {
				t.Errorf("a line that is not the player's own is marked own: %s", c.Line)
			}
		case SpeakerStory:
			neighbourCues++
			if contains(own, c.Line) {
				t.Errorf("the player's own line is marked as a neighbour: %s", c.Line)
			}
		}
	}
	// the session writes the host's introduction first, so every own
	// line but that one is the dog's
	if ownCues != len(own)-1 {
		t.Errorf("marked %d own lines, want %d", ownCues, len(own)-1)
	}
	if neighbourCues == 0 {
		t.Error("the neighbours are the lines that roll past, there must be some")
	}
	// the own segment is unbroken and last: nothing of anybody else's
	// arrives after the player's dog starts
	firstOwn := -1
	for i, c := range cues {
		if c.Speaker == SpeakerOwn && firstOwn < 0 {
			firstOwn = i
		}
		if firstOwn >= 0 && c.Speaker == SpeakerStory {
			t.Errorf("a neighbour line arrives after the own segment starts: %s", c.Line)
		}
	}
	// and a night with no own segment marks nothing own
	for _, c := range Broadcast(testPool(), nil, Ranger) {
		if c.Speaker == SpeakerOwn {
			t.Errorf("no own story was given, yet a line is marked own: %s", c.Line)
		}
	}
}
