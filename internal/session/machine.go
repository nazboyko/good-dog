package session

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nazboyko/good-dog/internal/visitor"
)

// The state machine is the single source of truth for one run: which
// day, which beat, what the player signaled, what happened with each
// visitor, and how the life ended. It is pure: Step takes a State and
// an Input and returns the next State or an error, no clock, no model,
// no network. Everything else renders it or persists it.

// Ending is one of the three honest endings, or empty while playing.
// None is a loss, per the game design.
type Ending string

const (
	EndingNone        Ending = ""
	EndingChosen      Ending = "chosen"
	EndingAnotherDog  Ending = "another_dog"
	EndingNobodyToday Ending = "nobody_today"
)

// Encounter is one visitor scene, appended to the bond history. Comfort
// is never stored as a number, only who came, every answer the player
// gave, the shape those answers made, and what the visit left behind.
type Encounter struct {
	Day       int             `json:"day"`
	Signals   []Vocalization  `json:"signals"`
	Archetype string          `json:"archetype"`
	Outcome   visitor.Outcome `json:"outcome"`
	Shape     visitor.Shape   `json:"shape"`
	// Arc names that shape in one line, so the history reads back
	Arc string `json:"arc"`
}

// State is everything the game knows about one run. It round trips
// through JSON unchanged, so a refresh resumes exactly here.
type State struct {
	Version int          `json:"version"`
	Day     int          `json:"day"`
	Beat    Beat         `json:"beat"`
	Signal  Vocalization `json:"signal,omitempty"`
	// Scene is the answers already given to the visitor at the gate,
	// cleared when they leave
	Scene     []Vocalization `json:"scene,omitempty"`
	Bond      []Encounter    `json:"bond"`
	Ending    Ending         `json:"ending,omitempty"`
	StartedAt time.Time      `json:"started_at"`
}

// bumped when the shape of State changes: version 3 made a visit four
// exchanges, so an encounter holds every answer and the shape they made
const stateVersion = 3

// Input is what the player can do. Advance moves on, Vocalize answers a
// visitor. The server decides what either means.
type Input struct {
	Kind   InputKind
	Signal Vocalization
}

type InputKind string

const (
	InputAdvance  InputKind = "advance"
	InputVocalize InputKind = "vocalize"
)

// Rails is the beat order for one day. The short run is the playtest
// prototype, one day then the reveal. The full run is three days, day
// three ending in adoption day, then the reveal.
type Rails struct {
	Days  int
	Beats []Beat
	// LastDay is the shape of the final day when it differs from the
	// rest. Adoption day ends on the meeting room and goes straight into
	// the reveal: a radio broadcast between the ending and the photo
	// would spend the one moment the whole run is built toward.
	LastDay []Beat
}

// beatsFor is the day's shape. Every day but the last is Beats.
func beatsFor(r Rails, day int) []Beat {
	if day >= r.Days && len(r.LastDay) > 0 {
		return r.LastDay
	}
	return r.Beats
}

// isVisit reports whether somebody is at the gate on this beat. The
// adoption scene runs on the visitor machinery, it is just longer and
// in a different room.
func isVisit(b Beat) bool { return b == BeatVisitor || b == BeatAdoption }

// exchangesFor is how many answers a scene is worth. The last visitor
// of the run gets more room than the ones before, because it is the
// scene the run has been building toward.
func exchangesFor(b Beat) int {
	if b == BeatAdoption {
		return visitor.AdoptionExchanges
	}
	return visitor.ExchangesPerScene
}

var (
	ShortRun = Rails{Days: 1, Beats: []Beat{BeatWake, BeatScent, BeatVisitor, BeatNight}}
	FullRun  = Rails{
		Days:    3,
		Beats:   []Beat{BeatWake, BeatScent, BeatVisitor, BeatVisitor, BeatNight},
		LastDay: []Beat{BeatWake, BeatScent, BeatAdoption},
	}
)

// NewState starts a run at day one, first beat.
func NewState(r Rails, now time.Time) State {
	return State{Version: stateVersion, Day: 1, Beat: r.Beats[0], Bond: []Encounter{}, StartedAt: now}
}

// Step is the whole rulebook. It never mutates its input.
func Step(r Rails, s State, in Input) (State, error) {
	if s.Beat == BeatDone {
		return s, fmt.Errorf("this life is over")
	}
	switch in.Kind {
	case InputVocalize:
		return vocalize(s, in.Signal)
	case InputAdvance:
		return advance(r, s)
	}
	return s, fmt.Errorf("unknown input %q", in.Kind)
}

func vocalize(s State, v Vocalization) (State, error) {
	if !isVisit(s.Beat) {
		return s, fmt.Errorf("nobody is here to hear that right now")
	}
	if !ValidVocalization(v) {
		return s, fmt.Errorf("unknown vocalization %q", v)
	}
	if s.Signal != "" {
		return s, fmt.Errorf("you already answered")
	}
	next := s
	next.Signal = v
	return next, nil
}

func advance(r Rails, s State) (State, error) {
	if isVisit(s.Beat) && s.Signal == "" {
		return s, fmt.Errorf("the visitor is waiting, choose a signal first")
	}
	next := s
	if s.Beat == BeatEpilogue {
		next.Beat = BeatDone
		return next, nil
	}
	i := beatIndex(r, s)
	if i < 0 {
		// a beat this build does not know, or a state from other rails
		return s, fmt.Errorf("this life is in a place this build does not know")
	}
	// a visit is several exchanges, so advancing inside one moves to the
	// next answer, and only the last one lets the day move on
	if isVisit(s.Beat) {
		scene := append(append([]Vocalization{}, s.Scene...), s.Signal)
		next.Signal = ""
		if len(scene) < exchangesFor(s.Beat) {
			next.Scene = scene
			return next, nil
		}
		a := s.VisitorAtGate()
		end := visitor.Close(a, visitorSignals(scene))
		if s.Beat == BeatAdoption {
			end = visitor.CloseAdoption(a, s.pastVisits(), visitorSignals(scene))
		}
		next.Bond = append(append([]Encounter{}, s.Bond...), Encounter{
			Day: s.Day, Signals: scene, Archetype: a.ID,
			Outcome: end.Outcome, Shape: end.Shape, Arc: end.Arc,
		})
		next.Scene = nil
	}
	today := beatsFor(r, s.Day)
	if i+1 < len(today) {
		next.Beat = today[i+1]
		return next, nil
	}
	// the day is over
	if s.Day < r.Days {
		next.Day = s.Day + 1
		next.Beat = beatsFor(r, next.Day)[0]
		return next, nil
	}
	// the last night ends the run, the ending is decided by the day
	next.Ending = decideEnding(next)
	next.Beat = BeatEpilogue
	return next, nil
}

// beatIndex finds where in the day's rails the state sits, or -1 when
// the state does not fit these rails. Repeated beat kinds (two visitors
// in one day) are told apart by how many encounters were logged today.
func beatIndex(r Rails, s State) int {
	today := beatsFor(r, s.Day)
	if s.Beat != BeatVisitor {
		for i, rb := range today {
			if rb == s.Beat {
				return i
			}
		}
		return -1
	}
	seenToday := s.visitorsToday()
	// the current visitor is the (seenToday+1)th visitor beat of the day
	nth := 0
	for i, rb := range today {
		if rb == BeatVisitor {
			if nth == seenToday {
				return i
			}
			nth++
		}
	}
	return -1
}

// visitorSignals converts a scene for the visitor package, which keeps
// its own vocabulary so it never imports the game state.
func visitorSignals(scene []Vocalization) []visitor.Signal {
	out := make([]visitor.Signal, len(scene))
	for i, v := range scene {
		out[i] = visitor.Signal(v)
	}
	return out
}

// visitorsToday is how many visitors this day has already met, which
// is also which visitor is at the gate right now.
func (s State) visitorsToday() int {
	n := 0
	for _, e := range s.Bond {
		if e.Day == s.Day {
			n++
		}
	}
	return n
}

// VisitorAtGate is who is standing there, derived from where the run
// is, so a resume always finds the same person.
// VisitorAtGate is who is here now. On adoption day it is the adopter,
// built from the run rather than picked from the list.
func (s State) VisitorAtGate() visitor.Archetype {
	if s.Beat == BeatAdoption {
		return visitor.Adopter(s.pastVisits())
	}
	return visitor.ArchetypeFor(s.Day, s.visitorsToday())
}

// SceneSoFar is every answer given to the visitor at the gate,
// including the one waiting to be acknowledged.
func (s State) SceneSoFar() []visitor.Signal {
	scene := s.Scene
	if s.Signal != "" {
		scene = append(append([]Vocalization{}, s.Scene...), s.Signal)
	}
	return visitorSignals(scene)
}

// Exchange is which answer of the visit this is, counting from one.
func (s State) Exchange() int { return len(s.Scene) + 1 }

// decideEnding reads how the run ended off the last encounter, which is
// the adoption scene. The three endings are the three things that
// really happen on an adoption day, and none of them is a loss:
// somebody chose you, somebody came and chose another dog, or nobody
// came today and tomorrow exists.
//
// A run with no adoption scene at all, which is the short rails, ends
// nobody today. That is the honest reading of a life with no adoption
// event in it, not a fallback.
func decideEnding(s State) Ending {
	for i := len(s.Bond) - 1; i >= 0; i-- {
		if s.Bond[i].Archetype != adopterID {
			continue
		}
		switch s.Bond[i].Outcome {
		case visitor.OutcomeAsked:
			return EndingChosen
		case visitor.OutcomeParted:
			// she stayed, she talked to the desk, and she left with the
			// dog she came to see. Somebody was chosen today, just not you
			return EndingAnotherDog
		default:
			return EndingNobodyToday
		}
	}
	return EndingNobodyToday
}

// adopterID is who the adoption scene puts in the meeting room. Kept
// here so the ending rule and the archetype cannot drift apart.
const adopterID = "adopter"

// pastVisits is the run so far in the terms the adoption scene needs:
// how each visit ended, in order.
func (s State) pastVisits() []visitor.Past {
	out := make([]visitor.Past, 0, len(s.Bond))
	for _, e := range s.Bond {
		if e.Archetype == adopterID {
			continue
		}
		out = append(out, visitor.Past{Outcome: e.Outcome})
	}
	return out
}

// Marshal and Unmarshal are the persistence contract. Unmarshal refuses
// a version it does not know, a resume must never guess.
func (s State) Marshal() ([]byte, error) { return json.Marshal(s) }

func UnmarshalState(raw []byte) (State, error) {
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return State{}, fmt.Errorf("state not json: %w", err)
	}
	if s.Version != stateVersion {
		return State{}, fmt.Errorf("state version %d, this build reads %d", s.Version, stateVersion)
	}
	if s.Day < 1 || !knownBeat(s.Beat) {
		return State{}, fmt.Errorf("state at day %d beat %q is not one this build knows", s.Day, s.Beat)
	}
	if s.Bond == nil {
		s.Bond = []Encounter{}
	}
	return s, nil
}

func knownBeat(b Beat) bool {
	switch b {
	case BeatWake, BeatScent, BeatVisitor, BeatAdoption, BeatNight, BeatEpilogue, BeatDone:
		return true
	}
	return false
}
