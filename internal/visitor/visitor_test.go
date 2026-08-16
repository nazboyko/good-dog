package visitor

import (
	"strings"
	"testing"
)

func TestEveryArchetypeHasAnOpinionAboutEverySignal(t *testing.T) {
	for _, a := range Archetypes {
		if len(a.Prefers) != len(Signals) {
			t.Errorf("%s has %d preferences, want %d", a.ID, len(a.Prefers), len(Signals))
		}
		for _, s := range Signals {
			if _, ok := a.Prefers[s]; !ok {
				// a missing entry would read as indifference by accident
				t.Errorf("%s has no opinion about %s", a.ID, s)
			}
		}
	}
}

// The band language is the whole tutorial. Each band must say a
// different thing a body does, or a player cannot read the room. Both
// tables have to hold that on their own, and neither may borrow a line
// from the other, or the two time scales stop being distinguishable.
func TestEachBandHasItsOwnBodyLanguage(t *testing.T) {
	tables := map[string]map[Band]string{"reaction": bodyTemplates, "settled": settledTemplates}
	for name, table := range tables {
		for _, a := range Archetypes {
			seen := map[string]Band{}
			for _, b := range Bands {
				line := a.render(table, b)
				if line == "" {
					t.Fatalf("%s has no %s line for band %s", a.ID, name, b)
				}
				if other, dup := seen[line]; dup {
					t.Errorf("%s %s: bands %s and %s share a line: %q", a.ID, name, other, b, line)
				}
				seen[line] = b
				if !strings.HasSuffix(line, ".") || strings.Contains(line, "—") {
					t.Errorf("%s %s: band %s line is not plain copy: %q", a.ID, name, b, line)
				}
			}
			if len(seen) != len(Bands) {
				t.Errorf("%s %s: %d distinct lines for %d bands", a.ID, name, len(seen), len(Bands))
			}
		}
	}
	for _, b := range Bands {
		if bodyTemplates[b] == settledTemplates[b] {
			t.Errorf("band %s reads the same after one answer as after four: %q", b, bodyTemplates[b])
		}
	}
}

// The narrator names one signal. From the second answer on, the body
// underneath it names the whole visit, so it has to be worded as where
// the visitor now stands. Otherwise a growl on the last exchange reads
// as the reason a hand is resting on the gate.
//
// Checked over every mixed scene, not scenes of one repeated answer:
// uniform scenes are the one family where the two can never disagree,
// which is exactly how this rule was lost once already.
func TestLaterExchangesSpeakForTheWholeVisit(t *testing.T) {
	reaction := map[string]bool{}
	for _, a := range Archetypes {
		for _, b := range Bands {
			reaction[a.render(bodyTemplates, b)] = true
		}
	}
	mixed := 0
	for _, a := range Archetypes {
		var walk func(scene []Signal)
		walk = func(scene []Signal) {
			if len(scene) > 0 {
				body := Body(a, scene)
				if body == "" {
					t.Fatalf("%s: scene %v has no body line", a.ID, scene)
				}
				if len(scene) == 1 && !reaction[body] {
					t.Errorf("%s: the first answer must read as a reaction, got %q", a.ID, body)
				}
				if len(scene) > 1 {
					if reaction[body] {
						t.Errorf("%s: scene %v of %d answers still reads as a reaction to one: %q",
							a.ID, scene, len(scene), body)
					}
					if !strings.Contains(body, "still") && !strings.Contains(body, "again") &&
						!strings.Contains(body, "has not") {
						t.Errorf("%s: scene %v does not read as where the visit stands: %q", a.ID, scene, body)
					}
					mixed++
				}
			}
			if len(scene) == ExchangesPerScene {
				return
			}
			for _, sig := range Signals {
				walk(append(scene, sig))
			}
		}
		walk(nil)
	}
	// 36 + 216 + 1296 scenes of two answers or more, for each archetype
	if want := 2 * (36 + 216 + 1296); mixed != want {
		t.Errorf("walked %d scenes of two answers or more, want %d", mixed, want)
	}
}

func TestBandsAreGraded(t *testing.T) {
	// more comfort never reads as less, and the whole range is used
	order := map[Band]int{}
	for i, b := range Bands {
		order[b] = i
	}
	last := -1
	for comfort := -4; comfort <= 4; comfort++ {
		got := order[bandFor(comfort)]
		if got < last {
			t.Errorf("comfort %d graded down to %s", comfort, bandFor(comfort))
		}
		last = got
	}
	if bandFor(-4) != BandDrifting || bandFor(4) != BandClose || bandFor(0) != BandWatching {
		t.Error("the ends and the middle of the scale must be reachable")
	}
}

// Honest matching: a visitor who came for another dog can never take
// this one home, no matter what the player does. That is not a failure
// state, it is the truth of a shelter afternoon.
func TestTheImpossibleVisitorCanNeverChooseButCanAlwaysPartWell(t *testing.T) {
	// every whole scene, four answers of one signal, plus mixed scenes
	best := OutcomeMovedOn
	for _, s := range Signals {
		scene := []Signal{s, s, s, s}
		e := Close(HereForAnother, scene)
		if e.Outcome == OutcomeAsked {
			t.Errorf("a scene of %s made an impossible visitor choose", s)
		}
		if e.Outcome == OutcomeParted {
			best = OutcomeParted
		}
	}
	if best != OutcomeParted {
		t.Error("parting well must be reachable, or the scene is not worth playing")
	}
	if e := Close(QuietSeeker, []Signal{Silence, Silence, Silence, Silence}); e.Outcome != OutcomeAsked {
		t.Errorf("a visitor who is looking must be reachable by a whole scene of the right answer, got %s", e.Outcome)
	}
}

// Four exchanges are what give the quiet visitor his whole range: one
// answer never decided a visit, and every rung is now reachable.
func TestAWholeSceneReachesEveryRung(t *testing.T) {
	for _, a := range Archetypes {
		reached := map[Band]bool{}
		for _, s := range Signals {
			for n := 1; n <= ExchangesPerScene; n++ {
				scene := make([]Signal, n)
				for i := range scene {
					scene[i] = s
				}
				band := bandFor(comfort(a, scene))
				reached[band] = true
			}
		}
		if len(reached) != len(Bands) {
			t.Errorf("%s only ever reaches %d of %d rungs across whole scenes: %v",
				a.ID, len(reached), len(Bands), reached)
		}
	}
	// the man who came for another dog can now be reached all the way
	band := bandFor(comfort(HereForAnother, []Signal{Whine, Silence, Whine, Silence}))
	if band != BandClose {
		t.Errorf("four warm answers must reach the top rung, got %s", band)
	}
}

// The shape is readable in retrospect, and it is a shape, never a score.
func TestSceneShapesReadTheMovement(t *testing.T) {
	cases := []struct {
		want  Shape
		scene []Signal
	}{
		{ShapeWarmed, []Signal{Silence, Silence, Silence, Silence}},
		{ShapeCooled, []Signal{Howl, Howl, Howl, Howl}},
		{ShapeSteady, []Signal{AlertBark, AlertBark, AlertBark, AlertBark}},
		// turned goes both ways and one line has to cover both, so both
		// are tested: up to warming and back, then down to distant and back
		{ShapeTurned, []Signal{AlertBark, Silence, Howl, AlertBark}},
		{ShapeTurned, []Signal{Howl, Silence, AlertBark, AlertBark}},
	}
	for _, c := range cases {
		if got := shapeOf(QuietSeeker, c.scene); got != c.want {
			t.Errorf("scene %v shaped %s, want %s", c.scene, got, c.want)
		}
	}
	// The visit is measured from where the visitor stood before the dog
	// made a sound, not from where the first answer left them. She spends
	// this whole scene one rung below the gate, so it cooled. Seeded from
	// the first answer instead it would read as steady, and the arc line
	// would tell the player nothing moved.
	if got := shapeOf(QuietSeeker, []Signal{Howl, AlertBark, AlertBark, AlertBark}); got != ShapeCooled {
		t.Errorf("a visit spent below where it started is cooled, got %s", got)
	}
	// every shape says something, in the present, with no verdict in it
	blame := []string{"wrong", "fail", "should have", "too bad", "unfortunately",
		"sorry", "missed", "lost", "mistake", "better", "worse", "good job", "well done"}
	for _, a := range Archetypes {
		seen := map[string]bool{}
		for sh := range arcLines {
			line := arcLine(a, sh)
			if line == "" {
				t.Fatalf("%s has no arc line for %s", a.ID, sh)
			}
			if seen[line] {
				t.Errorf("%s: two shapes share an arc line: %q", a.ID, line)
			}
			seen[line] = true
			lower := strings.ToLower(line)
			for _, word := range blame {
				if strings.Contains(lower, word) {
					t.Errorf("%s arc for %s carries a verdict %q: %s", a.ID, sh, word, line)
				}
			}
			for _, broken := range []string{"%!", "{", "}"} {
				if strings.Contains(line, broken) {
					t.Errorf("%s arc for %s rendered %q: %s", a.ID, sh, broken, line)
				}
			}
			for _, digit := range "0123456789" {
				if strings.ContainsRune(line, digit) {
					t.Errorf("%s arc for %s leaked a number: %s", a.ID, sh, line)
				}
			}
		}
	}
	if arcLine(QuietSeeker, Shape("nonsense")) != "" {
		t.Error("an unknown shape must render nothing, never a broken line")
	}
}

// The game never says wrong. Every line a player can read at parting
// says what happened, not how they did.
func TestPartingCopyIsConsequenceNeverFailure(t *testing.T) {
	// the arrival, the body and the parting all speak in the present,
	// so a player never has to wonder if the visitor is still there
	for _, a := range Archetypes {
		for _, o := range []Outcome{OutcomeAsked, OutcomeParted, OutcomeMovedOn} {
			line := parting(a, o, BandWarming)
			for _, pastTense := range []string{"stopped at", "said goodbye", "moved on down", "stayed at"} {
				if strings.Contains(line, pastTense) {
					t.Errorf("%s parting for %s slips into past tense: %s", a.ID, o, line)
				}
			}
		}
	}

	blame := []string{"wrong", "fail", "should have", "too bad", "unfortunately", "sorry",
		"missed", "lost", "mistake", "better luck", "try again"}
	for _, a := range Archetypes {
		for _, o := range []Outcome{OutcomeAsked, OutcomeParted, OutcomeMovedOn} {
			for _, b := range Bands {
				line := parting(a, o, b)
				if line == "" {
					t.Fatalf("%s has no parting line for %s", a.ID, o)
				}
				lower := strings.ToLower(line)
				for _, word := range blame {
					if strings.Contains(lower, word) {
						t.Errorf("%s parting for %s blames the player with %q: %s", a.ID, o, word, line)
					}
				}
				if strings.Contains(line, "—") || strings.Contains(line, "!") {
					t.Errorf("%s parting for %s is not the house voice: %s", a.ID, o, line)
				}
			}
		}
	}
	// The impossible visitor says why on every band a player can land
	// on, the cold ones included. Pairing an outcome with a band by hand
	// is how this was missed once: OutcomeParted with BandDrifting is
	// not a pair the game can produce, so it proved nothing about the
	// two bands where the reason was actually missing.
	for _, b := range Bands {
		line := parting(HereForAnother, OutcomeFor(HereForAnother, b), b)
		if !strings.Contains(line, "met last week") {
			t.Errorf("parting at band %s must say it was never about this dog: %s", b, line)
		}
	}
	// and staying with you reads differently from a polite goodbye
	warm := parting(HereForAnother, OutcomeParted, BandWarming)
	polite := parting(HereForAnother, OutcomeParted, BandWatching)
	if warm == polite {
		t.Error("parting well must not read the same as parting politely")
	}
	if !strings.Contains(warm, "late for that one") {
		t.Errorf("the warm parting must say what the dog changed: %s", warm)
	}
	// two men in one sentence must not share a pronoun, or the line
	// reads as him being late for himself
	if strings.Contains(warm, "late for him") {
		t.Errorf("the visitor and the dog he came for need different words: %s", warm)
	}
	// the visit ending without a result still lands somewhere, it never
	// leaves the player on the last line of a rejection
	for _, a := range Archetypes {
		movedOn := parting(a, OutcomeMovedOn, BandDistant)
		if !strings.Contains(movedOn, "chin back down") {
			t.Errorf("%s moving on must land back in the room: %s", a.ID, movedOn)
		}
	}
	// and one screen never says row three times
	screen := strings.ToLower(strings.Join(QuietSeeker.Arrival, " ") + " " +
		Body(QuietSeeker, []Signal{Howl}) + " " + parting(QuietSeeker, OutcomeMovedOn, BandDistant))
	if strings.Count(screen, "row") > 2 {
		t.Errorf("the word row is worn out on this screen: %s", screen)
	}
}

func TestArchetypeForIsStableAndCoversBoth(t *testing.T) {
	// the same position always finds the same person, so a resume does
	// not change who is at the gate
	if ArchetypeFor(1, 0).ID != ArchetypeFor(1, 0).ID {
		t.Error("archetype choice must be deterministic")
	}
	// the day has to matter, or every morning opens with the same face
	if ArchetypeFor(1, 0).ID == ArchetypeFor(2, 0).ID {
		t.Error("a new day must not open with the same visitor")
	}
	if ArchetypeFor(1, 0).ID != ArchetypeFor(1, 1).ID && ArchetypeFor(1, 1).ID != ArchetypeFor(2, 0).ID {
		t.Error("the order must roll forward across the day boundary")
	}
	seen := map[string]bool{}
	for day := 1; day <= 3; day++ {
		for nth := 0; nth < 2; nth++ {
			a := ArchetypeFor(day, nth)
			seen[a.ID] = true
			if _, ok := ByID(a.ID); !ok {
				t.Errorf("archetype %s is not findable by id", a.ID)
			}
		}
	}
	if len(seen) != len(Archetypes) {
		t.Errorf("a three day run must meet every archetype, met %d", len(seen))
	}
	if _, ok := ByID("nobody"); ok {
		t.Error("an unknown id must not resolve")
	}
}

func TestCloseIsPureAndReadable(t *testing.T) {
	for _, a := range Archetypes {
		for _, sig := range Signals {
			scene := []Signal{sig, sig, sig, sig}
			e := Close(a, scene)
			if e.Body == "" || e.Parting == "" || e.Outcome == "" || e.Shape == "" || e.Arc == "" {
				t.Fatalf("%s answering %s left something empty: %+v", a.ID, sig, e)
			}
			// the same scene twice reads the same, nothing accumulates
			if again := Close(a, scene); again != e {
				t.Errorf("%s answering %s is not pure", a.ID, sig)
			}
			// no number ever reaches the player
			for _, digit := range "0123456789" {
				if strings.ContainsRune(e.Body+e.Arc+e.Parting, digit) {
					t.Errorf("%s answering %s leaked a number: %+v", a.ID, sig, e)
				}
			}
		}
	}
}

// Every line a player can read is filled in. A leftover placeholder or
// a stray format verb would ship straight to the screen, and the tables
// are the only place one could come from.
func TestNoTemplateEverReachesAPlayerRaw(t *testing.T) {
	broken := []string{"{", "}", "%!", "%s", "%d", "%v"}
	for _, a := range Archetypes {
		lines := append([]string{}, a.Arrival...)
		for _, b := range Bands {
			lines = append(lines, a.render(bodyTemplates, b), a.render(settledTemplates, b))
			for _, o := range []Outcome{OutcomeAsked, OutcomeParted, OutcomeMovedOn} {
				lines = append(lines, parting(a, o, b))
			}
		}
		for sh := range arcLines {
			lines = append(lines, arcLine(a, sh))
		}
		for _, line := range lines {
			if line == "" {
				t.Errorf("%s has an empty line where copy should be", a.ID)
			}
			for _, bad := range broken {
				if strings.Contains(line, bad) {
					t.Errorf("%s renders %q raw: %s", a.ID, bad, line)
				}
			}
		}
		// the verb has to agree with the subject, or a they/them visitor
		// reads "they is still watching you"
		if a.Pronoun.Is == "" {
			t.Errorf("%s has no verb to agree with %q", a.ID, a.Pronoun.Subject)
		}
	}
}

// The player should feel where they are in a visit without being given
// a count. Every beat says something different, and none of it is a
// number or a fraction.
func TestOnwardNeverReadsTheSameTwiceAndNeverCounts(t *testing.T) {
	seen := map[string]int{}
	for nth := 1; nth <= ExchangesPerScene; nth++ {
		label := Onward(nth, ExchangesPerScene)
		if label == "" {
			t.Fatalf("exchange %d has no way forward", nth)
		}
		seen[label]++
		for _, digit := range "0123456789" {
			if strings.ContainsRune(label, digit) {
				t.Errorf("exchange %d counts at the player: %q", nth, label)
			}
		}
		for _, counting := range []string{" of ", "last", "final", "first", "next up"} {
			if strings.Contains(strings.ToLower(label), counting) {
				t.Errorf("exchange %d labels the position: %q", nth, label)
			}
		}
	}
	if len(seen) != ExchangesPerScene {
		t.Errorf("a visit of %d exchanges reads as %d different beats: %v", ExchangesPerScene, len(seen), seen)
	}
	if got := Onward(ExchangesPerScene, ExchangesPerScene); got != "the day goes on" {
		t.Errorf("the last exchange ends the visit, got %q", got)
	}
	// out of range on either side still gives a usable label
	if Onward(0, 4) == "" || Onward(99, 4) == "" {
		t.Error("there is always a way forward")
	}
}
