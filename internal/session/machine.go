package session

import (
	"encoding/json"
	"fmt"
	"time"
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

// Encounter is one visitor scene's outcome, appended to the bond
// history. Comfort is invisible as a number in the game, so this keeps
// only what the player did. What the visitor made of it is derived.
type Encounter struct {
	Day    int          `json:"day"`
	Signal Vocalization `json:"signal"`
}

// State is everything the game knows about one run. It round trips
// through JSON unchanged, so a refresh resumes exactly here.
type State struct {
	Version   int          `json:"version"`
	Day       int          `json:"day"`
	Beat      Beat         `json:"beat"`
	Signal    Vocalization `json:"signal,omitempty"`
	Bond      []Encounter  `json:"bond"`
	Ending    Ending       `json:"ending,omitempty"`
	StartedAt time.Time    `json:"started_at"`
}

const stateVersion = 1

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
}

var (
	ShortRun = Rails{Days: 1, Beats: []Beat{BeatWake, BeatScent, BeatVisitor, BeatNight}}
	FullRun  = Rails{Days: 3, Beats: []Beat{BeatWake, BeatScent, BeatVisitor, BeatVisitor, BeatNight}}
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
	if s.Beat != BeatVisitor {
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
	if s.Beat == BeatVisitor && s.Signal == "" {
		return s, fmt.Errorf("the visitor is waiting, choose a signal first")
	}
	next := s
	// leaving a visitor beat writes the encounter into the bond history
	if s.Beat == BeatVisitor {
		next.Bond = append(append([]Encounter{}, s.Bond...), Encounter{Day: s.Day, Signal: s.Signal})
		next.Signal = ""
	}
	if s.Beat == BeatEpilogue {
		next.Beat = BeatDone
		return next, nil
	}
	i := beatIndex(r, s)
	if i < 0 {
		// a beat this build does not know, or a state from other rails
		return s, fmt.Errorf("this life is in a place this build does not know")
	}
	if i+1 < len(r.Beats) {
		next.Beat = r.Beats[i+1]
		return next, nil
	}
	// the day is over
	if s.Day < r.Days {
		next.Day = s.Day + 1
		next.Beat = r.Beats[0]
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
	if s.Beat != BeatVisitor {
		for i, rb := range r.Beats {
			if rb == s.Beat {
				return i
			}
		}
		return -1
	}
	seenToday := 0
	for _, e := range s.Bond {
		if e.Day == s.Day {
			seenToday++
		}
	}
	// the current visitor is the (seenToday+1)th visitor beat of the day
	nth := 0
	for i, rb := range r.Beats {
		if rb == BeatVisitor {
			if nth == seenToday {
				return i
			}
			nth++
		}
	}
	return -1
}

// decideEnding is a placeholder rule until the adoption day scene and
// the comfort function land: with no visitor scoring yet, every run
// ends nobody today, the honest reading of a life with no adoption
// event. The other two endings are schema for that scene.
func decideEnding(s State) Ending {
	return EndingNobodyToday
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
	case BeatWake, BeatScent, BeatVisitor, BeatNight, BeatEpilogue, BeatDone:
		return true
	}
	return false
}
