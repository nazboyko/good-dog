package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nazboyko/good-dog/internal/visitor"
)

var adv = Input{Kind: InputAdvance}

func say(v Vocalization) Input { return Input{Kind: InputVocalize, Signal: v} }

// visit answers a whole scene with the same signal, the way a player
// clicking one button four times would.
// visit answers a whole scene, however long this one is. Adoption day
// is the same machinery with more exchanges.
func visit(t *testing.T, r Rails, s State, v Vocalization) State {
	t.Helper()
	beat := s.Beat
	for i := 0; i < exchangesFor(beat); i++ {
		if s.Beat != beat {
			t.Fatalf("exchange %d: scene changed from %s to %s", i, beat, s.Beat)
		}
		s = run(t, r, s, say(v), adv)
	}
	return s
}

// run drives a state through inputs, failing on any error.
func run(t *testing.T, r Rails, s State, inputs ...Input) State {
	t.Helper()
	for i, in := range inputs {
		next, err := Step(r, s, in)
		if err != nil {
			t.Fatalf("input %d (%s %s) from day %d %s: %v", i, in.Kind, in.Signal, s.Day, s.Beat, err)
		}
		s = next
	}
	return s
}

func TestShortRunIsThePrototypeRails(t *testing.T) {
	s := NewState(ShortRun, t0)
	want := []Beat{BeatWake, BeatScent, BeatVisitor, BeatNight, BeatEpilogue, BeatDone}
	for i, beat := range want {
		if s.Beat != beat || s.Day != 1 {
			t.Fatalf("step %d: day %d beat %s, want day 1 %s", i, s.Day, s.Beat, beat)
		}
		if beat == BeatDone {
			break
		}
		if beat == BeatVisitor {
			s = visit(t, ShortRun, s, Whine)
			continue
		}
		s = run(t, ShortRun, s, adv)
	}
	if s.Ending != EndingNobodyToday {
		t.Errorf("a run with no adoption event ends nobody today, got %q", s.Ending)
	}
	if len(s.Bond) != 1 || len(s.Bond[0].Signals) != visitor.ExchangesPerScene || s.Bond[0].Day != 1 {
		t.Errorf("the one visitor must be in the bond history with every answer: %+v", s.Bond)
	}
	if s.Bond[0].Arc == "" || s.Bond[0].Shape == "" {
		t.Errorf("the encounter must name the shape of the visit: %+v", s.Bond[0])
	}
}

// Two visitors a day for two days, then adoption day, which is one
// longer scene in the meeting room and no night: the ending goes
// straight into the reveal rather than putting a radio show between the
// peak of the run and the photo.
func TestFullRunWalksThreeDaysEndingOnAdoptionDay(t *testing.T) {
	s := NewState(FullRun, t0)
	visitors := 0
	for guard := 0; guard < 100 && s.Beat != BeatDone; guard++ {
		if isVisit(s.Beat) {
			s = visit(t, FullRun, s, Silence)
			visitors++
			continue
		}
		s = run(t, FullRun, s, adv)
	}
	if s.Beat != BeatDone {
		t.Fatalf("run did not finish, stuck at day %d %s", s.Day, s.Beat)
	}
	if visitors != 5 || len(s.Bond) != 5 {
		t.Errorf("two days of two visitors plus adoption day: met %d, bond %d", visitors, len(s.Bond))
	}
	days := map[int]int{}
	for _, e := range s.Bond {
		days[e.Day]++
	}
	if days[1] != 2 || days[2] != 2 || days[3] != 1 {
		t.Errorf("two encounters on days one and two, one on adoption day: %v", days)
	}
	if s.Ending == EndingNone {
		t.Error("a finished run must carry an ending")
	}
}

func TestVisitorRules(t *testing.T) {
	s := run(t, ShortRun, NewState(ShortRun, t0), adv, adv)
	if s.Beat != BeatVisitor {
		t.Fatalf("setup: %s", s.Beat)
	}
	if _, err := Step(ShortRun, s, adv); err == nil {
		t.Error("cannot leave a visitor without answering")
	}
	if _, err := Step(ShortRun, s, say(Vocalization("meow"))); err == nil {
		t.Error("unknown vocalization must be rejected")
	}
	s = run(t, ShortRun, s, say(Howl))
	if _, err := Step(ShortRun, s, say(Whine)); err == nil {
		t.Error("an answer is final, no second signal to the same exchange")
	}
	if _, err := Step(ShortRun, NewState(ShortRun, t0), say(Howl)); err == nil {
		t.Error("nobody to hear a signal at wake")
	}
}

func TestStepNeverMutatesItsInput(t *testing.T) {
	s := run(t, ShortRun, NewState(ShortRun, t0), adv, adv, say(Whine))
	before, _ := s.Marshal()
	if _, err := Step(ShortRun, s, adv); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Marshal()
	if string(before) != string(after) {
		t.Errorf("Step mutated its input:\n%s\n%s", before, after)
	}
}

// A scene already holding answers is the case that can alias, and
// marshalling the input will not catch it. Appending into the source
// slice's spare capacity writes at index len without changing len, so
// the input serializes identically and the test passes while one
// branch of the game quietly overwrites another.
//
// Stepping the same state twice is what sees it: if the second call
// writes through the shared backing array, it lands on top of the
// answer the first call recorded.
func TestSteppingOneStateTwiceKeepsBothAnswers(t *testing.T) {
	s := run(t, ShortRun, NewState(ShortRun, t0), adv, adv, say(Whine), adv, say(Silence), adv)
	if len(s.Scene) != 2 {
		t.Fatalf("setup: want a scene holding two answers, got %v", s.Scene)
	}
	// spare capacity is what makes the aliasing bug invisible, so make
	// sure there is some rather than hoping append left room
	roomy := make([]Vocalization, len(s.Scene), len(s.Scene)+4)
	copy(roomy, s.Scene)
	s.Scene = roomy

	howled := run(t, ShortRun, s, say(Howl), adv)
	growled := run(t, ShortRun, s, say(LowGrowl), adv)

	if got := howled.Scene[len(howled.Scene)-1]; got != Howl {
		t.Errorf("the second Step reached back into the first: scene ends %s, want howl", got)
	}
	if got := growled.Scene[len(growled.Scene)-1]; got != LowGrowl {
		t.Errorf("the second Step did not record its own answer: scene ends %s", got)
	}
	if len(s.Scene) != 2 {
		t.Errorf("Step grew the scene it was handed: %v", s.Scene)
	}
}

func TestDoneIsTerminal(t *testing.T) {
	s := visit(t, ShortRun, run(t, ShortRun, NewState(ShortRun, t0), adv, adv), Silence)
	s = run(t, ShortRun, s, adv, adv)
	if s.Beat != BeatDone {
		t.Fatalf("setup: %s", s.Beat)
	}
	if _, err := Step(ShortRun, s, adv); err == nil {
		t.Error("nothing moves past done")
	}
	if _, err := Step(ShortRun, s, say(Howl)); err == nil {
		t.Error("nothing to say past done")
	}
}

func TestStateRoundTripsThroughJSON(t *testing.T) {
	// day one: wake, scent, both visitors answered in full, night, then day two
	s := run(t, FullRun, NewState(FullRun, t0), adv, adv)
	s = visit(t, FullRun, s, PlayfulBark)
	s = visit(t, FullRun, s, Whine)
	s = run(t, FullRun, s, adv, adv)
	raw, err := s.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	back, err := UnmarshalState(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := back.Marshal()
	if string(again) != string(raw) {
		t.Errorf("round trip changed the state:\n%s\n%s", raw, again)
	}
	if back.Day != 2 || back.Beat != BeatScent || len(back.Bond) != 2 {
		t.Errorf("resumed at the wrong place: day %d %s bond %d", back.Day, back.Beat, len(back.Bond))
	}
	// and the resumed state keeps playing exactly like the original
	a := run(t, FullRun, s, adv, say(Silence), adv)
	b := run(t, FullRun, back, adv, say(Silence), adv)
	ra, _ := a.Marshal()
	rb, _ := b.Marshal()
	if string(ra) != string(rb) {
		t.Error("original and resumed states diverged under the same inputs")
	}
}

func TestUnmarshalRefusesUnknownVersion(t *testing.T) {
	raw := []byte(`{"version":99,"day":1,"beat":"wake","bond":[],"started_at":"2026-08-15T09:00:00Z"}`)
	if _, err := UnmarshalState(raw); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("a resume must never guess at an unknown version, got %v", err)
	}
	if _, err := UnmarshalState([]byte("not json")); err == nil {
		t.Error("bad json must fail")
	}
	// a state saved with no bond array resumes with an empty one, not nil
	s, err := UnmarshalState([]byte(`{"version":3,"day":1,"beat":"wake","started_at":"2026-08-15T09:00:00Z"}`))
	if err != nil || s.Bond == nil {
		t.Errorf("bond must never resume as nil: %v %+v", err, s)
	}
}

func TestEveryReachableStateReachesAnEnding(t *testing.T) {
	// breadth first over both rails: every input from every state either
	// errors or moves toward done, nothing dead ends and nothing loops
	for name, r := range map[string]Rails{"short": ShortRun, "full": FullRun} {
		seen := map[string]bool{}
		queue := []State{NewState(r, t0)}
		reachedDone := false
		for len(queue) > 0 && len(seen) < 5000 {
			s := queue[0]
			queue = queue[1:]
			key := stateKey(s)
			if seen[key] {
				continue
			}
			seen[key] = true
			if s.Beat == BeatDone {
				reachedDone = true
				continue
			}
			for _, in := range []Input{adv, say(Whine), say(Silence)} {
				if next, err := Step(r, s, in); err == nil {
					queue = append(queue, next)
				}
			}
		}
		if !reachedDone {
			t.Errorf("%s run: no path reaches done", name)
		}
		if len(seen) >= 5000 {
			t.Errorf("%s run: state space did not converge, something loops", name)
		}
	}
}

// stateKey collapses a state to what matters for reachability: the bond
// history and the scene in progress are summarized by length so the
// search space stays small.
//
// That is only safe while no transition reads what is inside them.
// beatIndex reads visitorsToday, which is Day plus len(Bond), and
// decideEnding ignores the bond entirely. The day decideEnding starts
// reading it, as its comment says it will, two states with the same key
// and different encounters will collapse into one and the search will
// prune the path to whichever ending it did not expand. It will still
// report success, which is the worst way for a proof to break. Add the
// field here first.
func stateKey(s State) string {
	b, _ := json.Marshal(struct {
		D  int
		B  Beat
		S  Vocalization
		Sc int
		N  int
		E  Ending
	}{s.Day, s.Beat, s.Signal, len(s.Scene), len(s.Bond), s.Ending})
	return string(b)
}

func TestAdvanceRefusesAStateTheseRailsDoNotKnow(t *testing.T) {
	// a state from the full run resumed under the short run: three
	// encounters today but the short run has one visitor beat
	s := NewState(FullRun, t0)
	s.Day = 1
	s.Beat = BeatVisitor
	s.Signal = Silence
	s.Bond = []Encounter{{Day: 1, Signals: []Vocalization{Whine}}, {Day: 1, Signals: []Vocalization{Howl}}, {Day: 1, Signals: []Vocalization{Silence}}}
	if _, err := Step(ShortRun, s, adv); err == nil {
		t.Fatal("a visitor beyond the rails must error, never restart the day")
	}
	unknown := NewState(ShortRun, t0)
	unknown.Beat = Beat("brunch")
	if _, err := Step(ShortRun, unknown, adv); err == nil {
		t.Fatal("an unknown beat must error, never restart the day")
	}
}

func TestUnmarshalRefusesUnknownBeatOrZeroDay(t *testing.T) {
	for _, raw := range []string{
		`{"version":3,"day":1,"beat":"brunch","bond":[],"started_at":"2026-08-15T09:00:00Z"}`,
		`{"version":3,"day":0,"beat":"wake","bond":[],"started_at":"2026-08-15T09:00:00Z"}`,
		`{"version":3,"day":1,"beat":"","bond":[],"started_at":"2026-08-15T09:00:00Z"}`,
	} {
		if _, err := UnmarshalState([]byte(raw)); err == nil {
			t.Errorf("must refuse: %s", raw)
		}
	}
}

// A three day run meets both kinds of visitor, and the one who could
// never take this dog home still leaves a real result behind.
func TestFullRunMeetsBothVisitorsAndRecordsHonestOutcomes(t *testing.T) {
	s := NewState(FullRun, t0)
	for guard := 0; guard < 100 && s.Beat != BeatDone; guard++ {
		if isVisit(s.Beat) {
			// silence, which every archetype reads as calm
			s = run(t, FullRun, s, say(Silence))
		}
		s = run(t, FullRun, s, adv)
	}
	// adoption day is the last thing that happens, and it decides the run
	if s.Ending == EndingNone {
		t.Error("a finished run must carry one of the three endings")
	}
	last := s.Bond[len(s.Bond)-1]
	if last.Archetype != adopterID || last.Day != 3 {
		t.Errorf("the last encounter of a run is the adoption scene: %+v", last)
	}
	if len(last.Signals) != visitor.AdoptionExchanges {
		t.Errorf("the last scene is %d exchanges, got %d", visitor.AdoptionExchanges, len(last.Signals))
	}
	met := map[string]visitor.Outcome{}
	for _, e := range s.Bond {
		if e.Archetype == "" || e.Outcome == "" || e.Arc == "" {
			t.Fatalf("every encounter must name who came, how it ended and its shape: %+v", e)
		}
		if len(e.Signals) == 0 {
			t.Fatalf("every encounter holds the whole scene: %+v", e)
		}
		met[e.Archetype] = e.Outcome
	}
	// every archetype from the row, plus the one who only ever comes on
	// the last day
	if len(met) != len(visitor.Archetypes)+1 {
		t.Errorf("a three day run must meet every archetype and the adopter, met %v", met)
	}
	if _, ok := met[adopterID]; !ok {
		t.Errorf("adoption day must put the adopter in the bond history: %v", met)
	}
	if met[visitor.QuietSeeker.ID] != visitor.OutcomeAsked {
		t.Errorf("silence should reach the visitor who is looking, got %q", met[visitor.QuietSeeker.ID])
	}
	if got := met[visitor.HereForAnother.ID]; got != visitor.OutcomeParted {
		t.Errorf("the visitor who came for another dog should part well, got %q", got)
	}
	// nothing the player did is recorded as a failure
	for _, e := range s.Bond {
		if e.Outcome == "failed" || e.Outcome == "lost" {
			t.Errorf("outcome vocabulary must never blame: %+v", e)
		}
	}
}

// Three honest endings, and every one of them reachable by playing.
// This is the piece that turns the run into a story with an end rather
// than a stop, so a run that can only ever end one way is a broken game
// however green the rest of the suite is.
func TestEveryEndingIsReachableByPlaying(t *testing.T) {
	// each signal played the whole way through a run, which is the
	// bluntest way to play and still has to reach all three
	found := map[Ending][]Vocalization{}
	for _, v := range vocalizations {
		s := NewState(FullRun, t0)
		for guard := 0; guard < 200 && s.Beat != BeatDone; guard++ {
			if isVisit(s.Beat) {
				s = run(t, FullRun, s, say(v))
			}
			s = run(t, FullRun, s, adv)
		}
		if s.Ending == EndingNone {
			t.Fatalf("answering %s all run left no ending", v)
		}
		found[s.Ending] = append(found[s.Ending], v)
	}
	for _, want := range []Ending{EndingChosen, EndingAnotherDog, EndingNobodyToday} {
		if len(found[want]) == 0 {
			t.Errorf("no way of playing reaches %q: %v", want, found)
		}
	}
}

// The ending is read off the adoption scene, not off the days before
// it. A warm week makes the last visitor easier to reach, it does not
// decide the run on its own.
func TestTheEndingComesFromTheAdoptionScene(t *testing.T) {
	// silence all week, then howling in the meeting room
	s := NewState(FullRun, t0)
	for guard := 0; guard < 200 && s.Beat != BeatAdoption && s.Beat != BeatDone; guard++ {
		if isVisit(s.Beat) {
			s = run(t, FullRun, s, say(Silence))
		}
		s = run(t, FullRun, s, adv)
	}
	if s.Beat != BeatAdoption {
		t.Fatalf("never reached adoption day, stuck at day %d %s", s.Day, s.Beat)
	}
	warmWeek := s
	for guard := 0; guard < 50 && s.Beat != BeatDone; guard++ {
		if isVisit(s.Beat) {
			s = run(t, FullRun, s, say(Howl))
		}
		s = run(t, FullRun, s, adv)
	}
	howled := s.Ending

	// the same warm week, met with silence in the meeting room
	s = warmWeek
	for guard := 0; guard < 50 && s.Beat != BeatDone; guard++ {
		if isVisit(s.Beat) {
			s = run(t, FullRun, s, say(Silence))
		}
		s = run(t, FullRun, s, adv)
	}
	if howled == s.Ending {
		t.Errorf("the last scene has to matter: howling and silence both ended %q", howled)
	}
}

// No ending is a loss, and none of them is a verdict on the player.
func TestNoEndingIsALoss(t *testing.T) {
	for _, e := range []Ending{EndingChosen, EndingAnotherDog, EndingNobodyToday} {
		if e == EndingNone {
			t.Error("the empty ending is not one of the three")
		}
		if strings.Contains(string(e), "fail") || strings.Contains(string(e), "lost") {
			t.Errorf("an ending named like a loss: %q", e)
		}
	}
}
