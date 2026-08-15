package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nazboyko/good-dog/internal/visitor"
)

var adv = Input{Kind: InputAdvance}

func say(v Vocalization) Input { return Input{Kind: InputVocalize, Signal: v} }

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
			s = run(t, ShortRun, s, say(Whine))
		}
		s = run(t, ShortRun, s, adv)
	}
	if s.Ending != EndingNobodyToday {
		t.Errorf("a run with no adoption event ends nobody today, got %q", s.Ending)
	}
	if len(s.Bond) != 1 || s.Bond[0].Signal != Whine || s.Bond[0].Day != 1 {
		t.Errorf("the one visitor must be in the bond history: %+v", s.Bond)
	}
}

func TestFullRunWalksThreeDaysAndTwoVisitorsADay(t *testing.T) {
	s := NewState(FullRun, t0)
	visitors := 0
	for guard := 0; guard < 100 && s.Beat != BeatDone; guard++ {
		if s.Beat == BeatVisitor {
			s = run(t, FullRun, s, say(Silence))
			visitors++
		}
		s = run(t, FullRun, s, adv)
	}
	if s.Beat != BeatDone {
		t.Fatalf("run did not finish, stuck at day %d %s", s.Day, s.Beat)
	}
	if visitors != 6 || len(s.Bond) != 6 {
		t.Errorf("three days times two visitors: met %d, bond %d", visitors, len(s.Bond))
	}
	days := map[int]int{}
	for _, e := range s.Bond {
		days[e.Day]++
	}
	if days[1] != 2 || days[2] != 2 || days[3] != 2 {
		t.Errorf("bond history must record two encounters per day: %v", days)
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
		t.Error("an answer is final, no second signal to the same visitor")
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

func TestDoneIsTerminal(t *testing.T) {
	s := run(t, ShortRun, NewState(ShortRun, t0), adv, adv, say(Silence), adv, adv, adv)
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
	s := run(t, FullRun, NewState(FullRun, t0), adv, adv, say(PlayfulBark), adv, say(Whine), adv, adv, adv)
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
	// day 1: wake, scent, visitor(bark), visitor(whine), night, then day 2 wake, scent
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
	s, err := UnmarshalState([]byte(`{"version":2,"day":1,"beat":"wake","started_at":"2026-08-15T09:00:00Z"}`))
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
// history is summarized by length so the search space stays small.
func stateKey(s State) string {
	b, _ := json.Marshal(struct {
		D int
		B Beat
		S Vocalization
		N int
		E Ending
	}{s.Day, s.Beat, s.Signal, len(s.Bond), s.Ending})
	return string(b)
}

func TestAdvanceRefusesAStateTheseRailsDoNotKnow(t *testing.T) {
	// a state from the full run resumed under the short run: three
	// encounters today but the short run has one visitor beat
	s := NewState(FullRun, t0)
	s.Day = 1
	s.Beat = BeatVisitor
	s.Signal = Silence
	s.Bond = []Encounter{{Day: 1, Signal: Whine}, {Day: 1, Signal: Howl}, {Day: 1, Signal: Silence}}
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
		`{"version":2,"day":1,"beat":"brunch","bond":[],"started_at":"2026-08-15T09:00:00Z"}`,
		`{"version":2,"day":0,"beat":"wake","bond":[],"started_at":"2026-08-15T09:00:00Z"}`,
		`{"version":2,"day":1,"beat":"","bond":[],"started_at":"2026-08-15T09:00:00Z"}`,
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
		if s.Beat == BeatVisitor {
			// silence, which both archetypes read as calm
			s = run(t, FullRun, s, say(Silence))
		}
		s = run(t, FullRun, s, adv)
	}
	met := map[string]visitor.Outcome{}
	for _, e := range s.Bond {
		if e.Archetype == "" || e.Outcome == "" {
			t.Fatalf("every encounter must name who came and how it ended: %+v", e)
		}
		met[e.Archetype] = e.Outcome
	}
	if len(met) != len(visitor.Archetypes) {
		t.Errorf("a three day run must meet every archetype, met %v", met)
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
