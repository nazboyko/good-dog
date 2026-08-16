// Package radio builds the night broadcast. It is pure: dogs and sheets
// in, a list of cues with offsets out. No clock, no network, no model.
// The server paces the cues, the client only plays them.
//
// The broadcast names other real dogs. That is the point of it: by
// morning the building is full of real animals rather than scenery. It
// is also why the copy is careful. A neighbour's story ends on
// something that is true about that dog and then their name and where
// they are, and it stops there. It never says a neighbour is real and
// never says they are still waiting. That beat belongs to the dog the
// player has been living as, at the epilogue, and spending it on a
// neighbour would fire the reveal early on the wrong animal.
package radio

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
)

// Speaker is who a cue belongs to, for the client to style.
type Speaker string

const (
	// SpeakerRanger is the host, the only invented voice in the game
	SpeakerRanger Speaker = "ranger"
	// SpeakerStory is a line about a real dog in another kennel
	SpeakerStory Speaker = "story"
	// SpeakerOwn is a line about the dog the player has been all day.
	// The night ends on these, so they stay on screen while the
	// neighbours' lines roll past above them.
	SpeakerOwn Speaker = "own"
)

// Cue is one line and when it lands, measured from the start of the
// broadcast. Offsets, never wall clock times, so a resume or a
// reconnect can play the same broadcast from its own start.
type Cue struct {
	At      time.Duration
	Speaker Speaker
	Line    string
	// Voice is who reads this line. Never serialized: the client is
	// handed a file to play, not a voice to choose.
	Voice Voice
	// Audio is the file this line is read from, empty when there is no
	// recording. Empty is a quiet cue, never a reason to skip the line:
	// the text is the night and the voice is on top of it.
	Audio string
}

// MarshalJSON writes the offset in milliseconds, because that is what
// the field is called and what a browser timer expects. A time.Duration
// marshals as nanoseconds by default, so the plain struct tag would
// have sent 1200000000 under a name that says ms and the client's
// fallback timer would have been set thirteen days out.
func (c Cue) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		At      int64   `json:"at_ms"`
		Speaker Speaker `json:"speaker"`
		Line    string  `json:"line"`
		Audio   string  `json:"audio,omitempty"`
	}{c.At.Milliseconds(), c.Speaker, c.Line, c.Audio})
}

// Neighbour is a real dog in the pool with its compiled sheet and the
// organization it actually belongs to. Nothing here is invented.
type Neighbour struct {
	Dog   animal.Animal
	Org   animal.Organization
	Sheet *dogsheet.DogSheet
}

// Pacing. Read out loud with the pauses in, then rounded.
const (
	openAt   = 1200 * time.Millisecond
	afterOne = 2600 * time.Millisecond
	betweenL = 2200 * time.Millisecond
	beforeUp = 2400 * time.Millisecond
)

// The host's lines. A small rotation rather than three constants: once
// living another life is the point of the game, a night that opens and
// closes with the same sentence every time turns Ranger into a jingle.
// Same voice, same tone, different words.
//
// Still fixed strings, which is what lets the whole spoken half of the
// radio be recorded once and read off a disk cache forever. Nothing
// here names a dog: the moment a host line took a name it would be per
// dog and could not be a constant.
var (
	hostOpens = []string{
		"Shelter radio, and it is late.",
		"Shelter radio. Nobody here is going anywhere tonight.",
		"This is the late one. Shelter radio.",
		"Shelter radio, and the building has gone quiet.",
	}
	hostBridges = []string{
		"Lights are down. Here is who is still awake.",
		"Lights are down. Here is tonight.",
		"Everybody is bedded down. Here is who is still up.",
		"Here is who is still awake, then.",
	}
	// The bridge for a night that has a bell in it. Three of the four
	// above promise that everyone named is still in the row, and one of
	// them will not be. "Here is tonight" is the one that promises
	// nothing, so it is the only one a bell night may open with. Same
	// for the opener that says nobody is going anywhere.
	bellBridge = "Lights are down. Here is tonight."
	bellOpen   = "Shelter radio, and it is late."
	hostCloses = []string{
		"That is everyone tonight. Sleep if you can.",
		"That is everyone. Get some sleep.",
		"That is the lot of them. Goodnight.",
		"That is everyone tonight. Rest up.",
	}
)

// HostOpen, HostWhoIsUp and HostClose are the first of each rotation.
// Kept named because tests and the boot check refer to them, and
// because a night with a zero seed is still a real night.
const (
	HostOpen    = "Shelter radio, and it is late."
	HostWhoIsUp = "Lights are down. Here is who is still awake."
	HostClose   = "That is everyone tonight. Sleep if you can."
)

// HostLines is every line Ranger can say, across the whole rotation.
// This is the set that has to exist in the cache before a night can
// start, because any night may draw any of them.
func HostLines() []string {
	out := make([]string, 0, len(hostOpens)+len(hostBridges)+len(hostCloses))
	out = append(out, hostOpens...)
	out = append(out, hostBridges...)
	out = append(out, hostCloses...)
	return out
}

// pick chooses one line of a rotation for this night. Deterministic on
// the seed, because the stream and the client's fallback both build the
// same night from the same session and a night that changed between
// them would play two different broadcasts.
func pick(lines []string, seed int64) string {
	if len(lines) == 0 {
		return ""
	}
	i := seed % int64(len(lines))
	if i < 0 {
		i = -i
	}
	return lines[i]
}

// The button under the night. It is disabled while the broadcast runs,
// and a disabled button with no explanation reads as a bug to somebody
// who does not know why, so it says the reason instead of nothing. No
// count, no bar, no time remaining: just what is true.
const (
	StillOn = "the radio is still on"
	Sleep   = "sleep"
)

// Stories is how many neighbours one night holds. Three is the number
// the plan falls back to when the evening runs short, so it is the
// number to build first.
const Stories = 3

// Broadcast lays out the whole night. Cue offsets only ever grow.
//
// The neighbours go first and the player's own dog goes last, because
// the last story of the night is the one the player is lying in. Own
// carries its own rules: it is the only story allowed to say the dog is
// real, and it is built by the session, not here.
func Broadcast(neighbours []Neighbour, own []string, ownVoice Voice, seed int64) []Cue {
	var cues []Cue
	at := openAt
	voice := Ranger
	add := func(who Speaker, line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		cues = append(cues, Cue{At: at, Speaker: who, Line: line, Voice: voice})
		at += betweenL
	}

	// a night that will ring the bell cannot open by promising that
	// everyone is still here
	open, bridge := pick(hostOpens, seed), pick(hostBridges, seed)
	if hasBell(neighbours) {
		open, bridge = bellOpen, bellBridge
	}
	add(SpeakerRanger, open)
	at += afterOne - betweenL
	add(SpeakerRanger, bridge)

	for i, n := range neighbours {
		if i >= Stories {
			break
		}
		at += beforeUp - betweenL
		// the dog says the true thing about themselves, the host says
		// who they are and where. A dog announcing its own name and
		// shelter in the third person is a station ident, not a dog.
		if q := sentence(quirkOf(n)); q != "" {
			voice = VoiceFor(n.Dog, n.Sheet)
			add(SpeakerStory, q)
		}
		voice = Ranger
		add(SpeakerRanger, naming(n))
	}

	if len(own) > 0 {
		// no "and next" line here: the session's own opener is older,
		// more specific and points at the player, and the silence of the
		// gap does the handoff better than a third announcement would
		at += beforeUp - betweenL
		for i, line := range own {
			// the session writes the host's introduction first and the
			// dog's own lines after it
			who, voiceHere := SpeakerOwn, ownVoice
			if i == 0 {
				who, voiceHere = SpeakerRanger, Ranger
			}
			voice = voiceHere
			add(who, line)
		}
	}

	at += beforeUp - betweenL
	voice = Ranger
	add(SpeakerRanger, pick(hostCloses, seed))
	return cues
}

// quirkOf is the one short true thing this dog does, and it is what a
// neighbour says on the radio. The radio seed is deliberately not used
// here. It is written to the shape of a listing, "Meet NAME, a
// high-energy, affectionate companion who...", and three of those in a
// row at two in the morning is three advertisements. It also carries the
// loosest claims in the sheet. The quirk is the cadence a tired host
// actually has.
func quirkOf(n Neighbour) string {
	if len(n.Sheet.Quirks) == 0 {
		return ""
	}
	return n.Sheet.Quirks[0].Value
}

// sentence closes a line that the model returned as a fragment. Done at
// render, never by editing the stored fact.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") {
		return s
	}
	return s + "."
}

// naming is the last line of a neighbour's story: their real name and
// the real place they are. It is the end of the story on purpose.
//
// Unless they are not there any more. A dog whose own listing says she
// was adopted gets the bell instead: the same slot, the same voice, one
// quiet line saying she went home. It is the only place in the whole
// game the radio reports good news, and it is only ever said about a
// confirmed adoption, never about a listing that simply went quiet.
func naming(n Neighbour) string {
	if n.Dog.Status == animal.StatusAdoptedConfirmed {
		return "That was " + n.Dog.Name + ". " + subject(n.Dog.Sex) + " went home."
	}
	where := OrgShort(n.Org.Name)
	switch {
	case where != "" && n.Org.City != "":
		return "That is " + n.Dog.Name + ". " + where + ", " + n.Org.City + "."
	case where != "":
		return "That is " + n.Dog.Name + ". " + where + "."
	default:
		return "That is " + n.Dog.Name + "."
	}
}

// hasBell is whether tonight names a dog who went home.
func hasBell(neighbours []Neighbour) bool {
	for i, n := range neighbours {
		if i >= Stories {
			break
		}
		if n.Dog.Status == animal.StatusAdoptedConfirmed {
			return true
		}
	}
	return false
}

// subject is the pronoun the listing's own sex field gives, and they
// when it gives none. Never a guess about a real animal.
func subject(sex string) string {
	switch strings.ToLower(sex) {
	case "female":
		return "She"
	case "male":
		return "He"
	}
	return "They"
}

// OrgShort trims an organization name at its first comma, so a line mid
// sentence reads as a place and not as a legal entity.
func OrgShort(name string) string {
	if i := strings.Index(name, ","); i > 0 {
		return strings.TrimSpace(name[:i])
	}
	return strings.TrimSpace(name)
}

// Neighbours picks who is on tonight. Dogs at the player's own shelter
// are left out: their story would name the place, and the place is part
// of the reveal the player has not reached yet.
func Neighbours(pool []Neighbour, playing animal.Animal, playingOrg string) []Neighbour {
	eligible := make([]Neighbour, 0, len(pool))
	heard := map[string]int{}
	for _, n := range pool {
		if n.Dog.ID == playing.ID || n.Org.ID == playingOrg {
			continue
		}
		if n.Sheet == nil || n.Sheet.Default {
			// a canned sheet has nothing true to say about this dog
			continue
		}
		// shelters really are full of dogs called Bella, but two of them
		// in one broadcast reads as a bug rather than as the truth. The
		// one who went home wins the name: her line is the one line of
		// good news the night has, and a namesake must not silence it.
		key := strings.ToLower(n.Dog.Name)
		if prev, ok := heard[key]; ok {
			if n.Dog.Status == animal.StatusAdoptedConfirmed && eligible[prev].Dog.Status != animal.StatusAdoptedConfirmed {
				eligible[prev] = n
			}
			continue
		}
		heard[key] = len(eligible)
		eligible = append(eligible, n)
	}

	// Spread the night across shelters where the pool allows it. Every
	// story ends by naming a place, so three dogs from one shelter means
	// the same sentence three times and the line stops being heard. A
	// repeat is still better than an empty slot, so the second pass
	// fills whatever the first pass left.
	var out []Neighbour
	used := map[string]bool{}
	for _, n := range eligible {
		if len(out) == Stories {
			break
		}
		if used[n.Org.ID] {
			continue
		}
		used[n.Org.ID] = true
		out = append(out, n)
	}
	for _, n := range eligible {
		if len(out) == Stories {
			break
		}
		if !containsDog(out, n.Dog.ID) {
			out = append(out, n)
		}
	}
	// A dog who went home is read last among the neighbours, right
	// before the player's own story. Opening the night on a homecoming
	// and then walking down the row of dogs still there gets the shape
	// backwards. Nothing else marks it: no pause, no sound, no second
	// line.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Dog.Status != animal.StatusAdoptedConfirmed && out[j].Dog.Status == animal.StatusAdoptedConfirmed
	})
	return out
}

func containsDog(list []Neighbour, id string) bool {
	for _, n := range list {
		if n.Dog.ID == id {
			return true
		}
	}
	return false
}
