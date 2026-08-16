package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/visitor"
)

var (
	t0 = time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)

	testDog = animal.Animal{
		ID: "rsmn-a-9548", Name: "Venus", Breed: "Pit Bull Terrier mix", AgeGroup: "Adult",
		Sex: "Female", AgeText: "4Y/1M/1W", WeightText: "49 lbs", LongStay: true,
		LongStayEvidence: animal.LongStayPlacement,
		Description:      "Goofy. Lap pet. Timid / Shy.",
		PhotoLocal:       "cache/photos/rsmn-a-9548.jpg",
		ListingURL:       "https://new.shelterluv.com/embed/animal/RSMN-A-9548",
		OrgID:            "ruff-start-rescue",
	}
	testOrg = animal.Organization{ID: "ruff-start-rescue", Name: "Ruff Start Rescue", City: "Princeton", State: "MN"}
)

func testSheet() *dogsheet.DogSheet {
	return &dogsheet.DogSheet{
		AnimalID: testDog.ID,
		Facts: []dogsheet.VerifiedFact{
			{ID: "f1", Value: "Venus", Source: "field:name"},
			{ID: "f2", Value: "Goofy.", Source: "description"},
			{ID: "f3", Value: "Lap pet.", Source: "description"},
		},
		Quirks:    []dogsheet.NarrativeInference{{Value: "Loves to snuggle right into your lap.", DerivedFrom: []string{"f3"}}},
		Movement:  dogsheet.NarrativeInference{Value: "moves with high energy, then curls up on a lap", DerivedFrom: []string{"f3"}},
		Voice:     dogsheet.NarrativeInference{Value: "quiet until something matters"},
		RadioSeed: dogsheet.NarrativeInference{Value: "Here is Venus, a smart and goofy adventurer.", DerivedFrom: []string{"f1", "f2"}},
	}
}

// playVisit answers a whole visit through the session, the way a player
// clicking one button four times would.
func playVisit(t *testing.T, s *Session, v Vocalization) {
	t.Helper()
	for i := 0; i < visitor.ExchangesPerScene; i++ {
		if err := s.Vocalize(v); err != nil {
			t.Fatalf("exchange %d: %v", i, err)
		}
		if err := s.Advance(); err != nil {
			t.Fatalf("exchange %d: %v", i, err)
		}
	}
}

func newTestSession() *Session {
	return New(testDog, testOrg, testSheet(), ShortRun, t0)
}

func TestRailsRunInOrder(t *testing.T) {
	s := newTestSession()
	want := []Beat{BeatWake, BeatScent, BeatVisitor, BeatNight, BeatEpilogue, BeatDone}
	for i, beat := range want {
		if s.Beat() != beat {
			t.Fatalf("step %d: beat = %s, want %s", i, s.Beat(), beat)
		}
		if beat == BeatVisitor {
			playVisit(t, s, Whine)
			continue
		}
		if beat == BeatDone {
			if err := s.Advance(); err == nil {
				t.Fatal("advancing past done must fail")
			}
			break
		}
		if err := s.Advance(); err != nil {
			t.Fatalf("advance from %s: %v", beat, err)
		}
	}
}

func TestVisitorWaitsForASignal(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	if s.Beat() != BeatVisitor {
		t.Fatalf("setup: beat %s", s.Beat())
	}
	if err := s.Advance(); err == nil {
		t.Fatal("visitor beat must not advance before a signal")
	}
	if err := s.Vocalize(Vocalization("meow")); err == nil {
		t.Fatal("unknown vocalization must be rejected")
	}
	if err := s.Vocalize(Silence); err != nil {
		t.Fatal(err)
	}
	v := s.View(t0)
	if v.Visitor == nil || v.Visitor.Mismatch == nil {
		t.Fatal("after a signal the view carries the mismatch narrator")
	}
	if v.Visitor.Mismatch.Meant == "" || v.Visitor.Mismatch.Heard == "" {
		t.Error("both narrator lines must be present")
	}
	if err := s.Advance(); err != nil {
		t.Fatalf("advance after signal: %v", err)
	}
}

func TestVocalizeOutsideVisitorBeatFails(t *testing.T) {
	s := newTestSession()
	if err := s.Vocalize(PlayfulBark); err == nil {
		t.Fatal("no visitor is present at wake")
	}
}

// The reveal rule: nothing before the epilogue may carry the photo or
// the listing, in any field, under any key.
func TestPhotoAndListingNeverLeakBeforeEpilogue(t *testing.T) {
	s := newTestSession()
	for s.Beat() != BeatEpilogue {
		raw, err := json.Marshal(s.View(t0))
		if err != nil {
			t.Fatal(err)
		}
		body := string(raw)
		for _, secret := range []string{"photo", ".jpg", "shelterluv", "listing", "Ruff Start", "long_stay", "49 lbs", "4Y/1M", "four years", "Minnesota"} {
			if strings.Contains(body, secret) {
				t.Fatalf("beat %s leaks %q: %s", s.Beat(), secret, body)
			}
		}
		if s.Beat() == BeatVisitor {
			s.Vocalize(Silence)
		}
		if err := s.Advance(); err != nil {
			t.Fatal(err)
		}
	}
	e := s.View(t0.Add(38 * time.Minute)).Epilogue
	if e == nil {
		t.Fatal("epilogue view missing at the epilogue beat")
	}
	if e.PhotoURL == "" || e.ListingURL == "" || e.OrgName == "" {
		t.Errorf("epilogue must carry photo, listing and org: %+v", e)
	}
	if !e.LongStay || e.MinutesPlayed != 38 {
		t.Errorf("epilogue facts wrong: long_stay=%v minutes=%d", e.LongStay, e.MinutesPlayed)
	}
	// the moment speaks plain words, the record keeps the listing's own
	if e.AgeWords != "four years old" || e.OrgState != "Minnesota" {
		t.Errorf("reveal must use plain words: age %q state %q", e.AgeWords, e.OrgState)
	}
	if e.Listing.AgeText != "4Y/1M/1W" || e.Listing.WeightText != "49 lbs" || len(e.Listing.Quotes) != 2 || e.Listing.Description == "" {
		t.Errorf("listing record must keep the verbatim strings: %+v", e.Listing)
	}
}

func TestStateName(t *testing.T) {
	if StateName("MN") != "Minnesota" || StateName("mn") != "Minnesota" {
		t.Error("postal codes spell out")
	}
	if StateName("Ontario") != "Ontario" || StateName("") != "" {
		t.Error("unknown values pass through untouched")
	}
}

func TestViewCarriesOnlyTheBeatsOwnData(t *testing.T) {
	s := newTestSession()
	v := s.View(t0)
	if v.Scent != nil || v.Visitor != nil || v.Night != nil || v.Epilogue != nil {
		t.Errorf("wake view must carry nothing but the basics: %+v", v)
	}
	if v.Name != "Venus" || v.AgeGroup != "Adult" || v.Breed == "" {
		t.Errorf("the player knows only name, age group and breed: %+v", v)
	}
	s.Advance()
	if s.View(t0).Scent == nil {
		t.Error("scent beat must carry the movement line")
	}
}

func TestRadioStoryGroundedAndEndsWithName(t *testing.T) {
	story := RadioStory(testDog, testSheet())
	joined := strings.Join(story, " ")
	for _, want := range []string{"Here is Venus", "snuggle", "She has been waiting a while.", "Her name is Venus.", "still here"} {
		if !strings.Contains(joined, want) {
			t.Errorf("story missing %q:\n%s", want, strings.Join(story, "\n"))
		}
	}
	// the shelter is part of the reveal, the radio never names it early
	if strings.Contains(joined, "Ruff Start") || strings.Contains(joined, "Princeton") {
		t.Errorf("radio must not name the place before the epilogue: %s", joined)
	}
	for _, line := range story {
		if strings.Contains(line, "\u2014") {
			t.Errorf("no em dashes in radio lines: %q", line)
		}
		if strings.Contains(strings.ToLower(line), "adopt") {
			t.Errorf("radio never talks adoption: %q", line)
		}
	}
	if !strings.HasPrefix(story[len(story)-1], "She is real") {
		t.Errorf("last line must be the real name and place beat, got %q", story[len(story)-1])
	}
}

func TestRadioStoryPronounsFollowTheListing(t *testing.T) {
	male := testDog
	male.Sex = "Male"
	story := strings.Join(RadioStory(male, testSheet()), " ")
	if !strings.Contains(story, "His name is") || strings.Contains(story, "Her name") {
		t.Errorf("male listing must read his: %s", story)
	}
	unknown := testDog
	unknown.Sex = ""
	story = strings.Join(RadioStory(unknown, testSheet()), " ")
	for _, want := range []string{"Their name is", "They have been waiting", "They are real, and they are still here."} {
		if !strings.Contains(story, want) {
			t.Errorf("unlisted sex must read they with matching verbs, missing %q: %s", want, story)
		}
	}
	noLongStay := testDog
	noLongStay.LongStay = false
	if strings.Contains(strings.Join(RadioStory(noLongStay, testSheet()), " "), "waiting a while") {
		t.Error("the waiting line must appear only when the long stay fact is present")
	}
}

func TestGeneratedLinesNamingTheShelterAreHeldBackBeforeReveal(t *testing.T) {
	sheet := testSheet()
	sheet.RadioSeed.Value = "Here is Venus, the pride of Ruff Start Rescue."
	sheet.Movement.Value = "She bounces around Princeton like she owns it."
	s := New(testDog, testOrg, sheet, ShortRun, t0)

	s.Advance()
	if got := s.View(t0).Scent.Movement; got != "" {
		t.Errorf("movement naming the city must be dropped before the reveal, got %q", got)
	}
	s.Advance()
	playVisit(t, s, Silence)
	for _, line := range s.View(t0).Night.Story {
		if strings.Contains(line, "Ruff Start") || strings.Contains(line, "Princeton") {
			t.Errorf("radio line names the shelter before the reveal: %q", line)
		}
	}
	// the name still lands, the story just loses the spoiling seed
	if !strings.Contains(strings.Join(s.View(t0).Night.Story, " "), "Her name is Venus.") {
		t.Error("holding back a line must not lose the name beat")
	}
}

func TestNarrateEveryVocalizationAndFallback(t *testing.T) {
	for _, v := range vocalizations {
		m := Narrate(v)
		if m.Meant == "" || m.Heard == "" {
			t.Errorf("%s has an empty narrator line", v)
		}
		// the narrator never knows the dog, so it may never guess a pronoun
		for _, line := range []string{" " + m.Meant + " ", " " + m.Heard + " "} {
			for _, pronoun := range []string{" she ", " he ", " her ", " his ", " him "} {
				if strings.Contains(line, pronoun) {
					t.Errorf("%s narrator line guesses a pronoun: %q", v, line)
				}
			}
		}
	}
	if Narrate(Vocalization("meow")) != Narrate(Silence) {
		t.Error("unknown signals narrate as silence")
	}
}

func TestStoreMemoryOnly(t *testing.T) {
	st := NewStore(nil, nil)
	s := newTestSession()
	if err := st.Put(context.Background(), s, t0); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(context.Background(), s.ID)
	if err != nil || got != s {
		t.Fatal("store must return the same session")
	}
	if _, err := st.Get(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id must be ErrNotFound, got %v", err)
	}
}

func TestStoreResumesFromTheDBAfterARestart(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	// the loader rebuilds a session from a row, the way the server does
	load := func(ctx context.Context, row Row) (*Session, error) {
		rails, ok := RailsByName(row.Rails)
		if !ok {
			return nil, errors.New("unknown rails")
		}
		return Resume(row.ID, testDog, testOrg, testSheet(), rails, row.State), nil
	}
	st := NewStore(db, load)
	s := newTestSession()
	if err := st.Put(ctx, s, t0); err != nil {
		t.Fatal(err)
	}
	s.Advance()
	s.Advance()
	s.Vocalize(Whine)
	if err := st.Put(ctx, s, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(s.View(t0))

	// simulated restart: the cache is gone, the DB is not
	st.forget(s.ID)
	back, err := st.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("resume after restart: %v", err)
	}
	if back == s {
		t.Fatal("test setup: the session must have been rebuilt, not cached")
	}
	after, _ := json.Marshal(back.View(t0))
	if string(before) != string(after) {
		t.Errorf("view after restart differs:\n%s\n%s", before, after)
	}
	// the visit resumes mid scene and finishes where it left off
	if err := back.Advance(); err != nil {
		t.Fatalf("resumed session must keep playing: %v", err)
	}
	if back.Beat() != BeatVisitor || len(back.State().Scene) != 1 {
		t.Errorf("a resume mid visit keeps the answers already given: %s %v", back.Beat(), back.State().Scene)
	}
	for i := 0; i < visitor.ExchangesPerScene-1; i++ {
		back.Vocalize(Silence)
		back.Advance()
	}
	if back.Beat() != BeatNight {
		t.Errorf("finishing the resumed visit moves the day on, at %s", back.Beat())
	}
	// and a second Get returns the same rebuilt instance, not another copy
	again, _ := st.Get(ctx, s.ID)
	if again != back {
		t.Error("the rebuilt session must be cached")
	}
}

func TestStoreMissWhenTheLoaderCannotRebuild(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	failing := func(ctx context.Context, row Row) (*Session, error) { return nil, errors.New("dog left the pool") }
	st := NewStore(db, failing)
	s := newTestSession()
	st.Put(ctx, s, t0)
	st.forget(s.ID)
	if _, err := st.Get(ctx, s.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("a session that cannot be rebuilt must be a clean miss, got %v", err)
	}
}

// State promises a copy a caller can scribble on. A shallow copy of the
// struct always passes a scalar check, because the struct really is
// copied. The slices inside it are what a shallow copy hands straight
// back, so this writes through every one of them.
func TestStateHandsOutNothingLive(t *testing.T) {
	s := New(testDog, testOrg, testSheet(), FullRun, t0)
	toVisitor := func() {
		for s.Beat() != BeatVisitor {
			if err := s.Advance(); err != nil {
				t.Fatalf("setup: %v", err)
			}
		}
	}
	toVisitor()
	playVisit(t, s, Whine)
	// a second visit, one answer in and taken, so a live scene and a
	// closed encounter are both on the state at once
	toVisitor()
	if err := s.Vocalize(Silence); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := s.Advance(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	loose := s.State()
	if len(loose.Bond) == 0 || len(loose.Bond[0].Signals) == 0 || len(loose.Scene) == 0 {
		t.Fatalf("setup: need a closed encounter and a live scene, got %+v", loose)
	}
	loose.Bond[0].Archetype = "tampered"
	loose.Bond[0].Signals[0] = Howl
	loose.Scene[0] = Howl

	live := s.State()
	if live.Bond[0].Archetype == "tampered" {
		t.Error("State handed out the live bond slice")
	}
	if live.Bond[0].Signals[0] == Howl {
		t.Error("State handed out the live signals inside an encounter")
	}
	if live.Scene[0] == Howl {
		t.Error("State handed out the live scene")
	}
}

func TestResumeRebuildsTheSameLife(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	s.Vocalize(Whine)
	raw, err := s.State().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	st, err := UnmarshalState(raw)
	if err != nil {
		t.Fatal(err)
	}
	back := Resume(s.ID, testDog, testOrg, testSheet(), ShortRun, st)
	if back.ID != s.ID || back.Beat() != BeatVisitor {
		t.Fatalf("resume landed wrong: id %s beat %s", back.ID, back.Beat())
	}
	a, b := s.View(t0), back.View(t0)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Errorf("resumed view differs:\n%s\n%s", ja, jb)
	}
	if err := back.Advance(); err != nil {
		t.Fatalf("resumed session must keep playing: %v", err)
	}
	if back.View(t0).Day != 1 || back.Beat() != BeatVisitor {
		t.Errorf("after resume and advance: day %d beat %s", back.View(t0).Day, back.Beat())
	}
	// and the rest of the visit still closes it
	for i := 0; i < visitor.ExchangesPerScene-1; i++ {
		back.Vocalize(Whine)
		back.Advance()
	}
	if back.Beat() != BeatNight || len(back.State().Bond) != 1 {
		t.Errorf("the resumed visit must close: %s %+v", back.Beat(), back.State().Bond)
	}
}

func TestViewCarriesDayAndEndingAtTheEnd(t *testing.T) {
	s := newTestSession()
	for s.Beat() != BeatEpilogue {
		if s.Beat() == BeatVisitor {
			s.Vocalize(Silence)
		}
		s.Advance()
	}
	v := s.View(t0)
	if v.Day != 1 || v.Ending != EndingNobodyToday {
		t.Errorf("epilogue view: day %d ending %q", v.Day, v.Ending)
	}
	if newTestSession().View(t0).Ending != EndingNone {
		t.Error("no ending before the run ends")
	}
}

// The visitor package keeps its own signal vocabulary so it never has
// to import the game state. This is the guard that the two never drift.
func TestVocalizationsAndVisitorSignalsNeverDrift(t *testing.T) {
	if len(vocalizations) != len(visitor.Signals) {
		t.Fatalf("%d vocalizations, %d visitor signals", len(vocalizations), len(visitor.Signals))
	}
	for _, v := range vocalizations {
		found := false
		for _, s := range visitor.Signals {
			if string(s) == string(v) {
				found = true
			}
		}
		if !found {
			t.Errorf("vocalization %s has no visitor signal", v)
		}
		for _, a := range visitor.Archetypes {
			if _, ok := a.Prefers[visitor.Signal(v)]; !ok {
				t.Errorf("archetype %s has no opinion about %s", a.ID, v)
			}
		}
	}
}

func TestVisitorViewCarriesBodyNotANumber(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	v := s.View(t0)
	if v.Visitor == nil || len(v.Visitor.Arrival) == 0 || v.Visitor.HeardLabel == "" {
		t.Fatalf("the visitor must arrive with lines and a narrator label: %+v", v.Visitor)
	}
	if v.Visitor.Body != "" || v.Visitor.Parting != "" {
		t.Error("nothing is read off a visitor before the player answers")
	}
	if v.Visitor.Exchange != 1 || v.Visitor.Exchanges != visitor.ExchangesPerScene {
		t.Errorf("the view must say which answer this is: %+v", v.Visitor)
	}
	if err := s.Vocalize(Silence); err != nil {
		t.Fatal(err)
	}
	v = s.View(t0)
	if v.Visitor.Body == "" {
		t.Fatalf("after an answer the visitor's body must read: %+v", v.Visitor)
	}
	// the visit only reads back its shape once it is over
	if v.Visitor.Arc != "" || v.Visitor.Parting != "" {
		t.Errorf("the first answer must not end the visit: %+v", v.Visitor)
	}
	for i := 1; i < visitor.ExchangesPerScene; i++ {
		s.Advance()
		s.Vocalize(Silence)
	}
	v = s.View(t0)
	if v.Visitor.Arc == "" || v.Visitor.Parting == "" {
		t.Fatalf("the last answer must read back the shape and the parting: %+v", v.Visitor)
	}
	// comfort never reaches the player as a number
	raw, _ := json.Marshal(v.Visitor)
	for _, word := range []string{"comfort", "score", "band"} {
		if strings.Contains(strings.ToLower(string(raw)), word) {
			t.Errorf("the visitor view leaks %q: %s", word, raw)
		}
	}
}

func TestEncounterRecordsWhoCameAndHowItEnded(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	who := s.State().VisitorAtGate()
	// a visit is four exchanges, and only the last one ends it
	for i := 0; i < visitor.ExchangesPerScene; i++ {
		if len(s.State().Bond) != 0 {
			t.Fatalf("the visit ended after %d exchanges, too early", i)
		}
		if err := s.Vocalize(Silence); err != nil {
			t.Fatal(err)
		}
		if err := s.Advance(); err != nil {
			t.Fatal(err)
		}
	}
	bond := s.State().Bond
	if len(bond) != 1 {
		t.Fatalf("one visit, one encounter: %+v", bond)
	}
	if bond[0].Archetype != who.ID || bond[0].Outcome == "" || len(bond[0].Signals) != visitor.ExchangesPerScene {
		t.Errorf("the encounter must name who came and every answer: %+v", bond[0])
	}
	// the shape reads back in one line, with no score in it
	if bond[0].Arc == "" || bond[0].Shape == "" {
		t.Errorf("the encounter must name the shape of the visit: %+v", bond[0])
	}
	if s.Beat() != BeatNight {
		t.Errorf("the day moves on after the visit, at %s", s.Beat())
	}
}

// The narrator and the visitor's body must never say opposite things.
// A signal the visitor heard badly can never leave them at their
// warmest, or the player learns the narrator cannot be trusted.
//
// This is the first answer of a visit, the only exchange where the two
// lines are about the same thing. From the second answer on the
// narrator still names one signal while the body holds the whole visit,
// and the body says so in its own words. The visitor package owns that
// half of the rule, because it is the one that can see both tables.
func TestNarratorAndBodyNeverContradict(t *testing.T) {
	// how each heard line reads, written down once, so a new signal or
	// a new archetype cannot quietly break the pairing
	heardWarmth := map[Vocalization]int{
		PlayfulBark: -1, // too loud, too sudden
		AlertBark:   0,  // something out there, not me
		Whine:       0,  // not sure about me
		LowGrowl:    -1, // maybe not this one
		Howl:        -1, // a lot, all at once
		Silence:     1,  // calm, watching me
	}
	if len(heardWarmth) != len(vocalizations) {
		t.Fatalf("every signal needs a reading, have %d of %d", len(heardWarmth), len(vocalizations))
	}
	// the warmest two lines in the body vocabulary, named here rather
	// than derived, so this still fails if the tables are reordered
	warmest := []string{"crouches down to your level", "puts a hand flat against the gate"}
	for _, a := range visitor.Archetypes {
		for _, v := range vocalizations {
			if heardWarmth[v] >= 0 {
				continue
			}
			body := visitor.Body(a, []visitor.Signal{visitor.Signal(v)})
			for _, phrase := range warmest {
				if strings.Contains(body, phrase) {
					t.Errorf("%s: the narrator says %q was heard as %q, then the body says %q",
						a.ID, v, Narrate(v).Heard, body)
				}
			}
		}
	}
}

// The visitor screen has to read as a conversation growing, not the
// same frame reprinted, so the view carries the exchanges already past.
//
// The first version of this test named that rule in its comment and
// then asserted only that four answers produced four entries, which the
// code would satisfy by printing one sentence four times. It did. The
// assertions below are the ones the comment was always promising.
func TestTheVisitCarriesTheExchangesAlreadyPast(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	if s.Beat() != BeatVisitor {
		t.Fatalf("setup: %s", s.Beat())
	}
	said := []Vocalization{Silence, Howl, Whine, LowGrowl}
	for i, v := range said {
		if err := s.Vocalize(v); err != nil {
			t.Fatal(err)
		}
		vv := s.View(t0).Visitor
		if vv == nil {
			t.Fatal("no visitor on the view")
		}
		// the column holds what is already past, never the loud line
		if len(vv.Settled) > i {
			t.Errorf("after %d answers the column shows %d rows", i+1, len(vv.Settled))
		}
		for _, row := range vv.Settled {
			if row == vv.Body {
				t.Errorf("the loud line is repeated in the column: %q", row)
			}
			// the column is moments, so it never claims continuity
			for _, claim := range []string{"still", "again", "has not"} {
				if strings.Contains(strings.ToLower(row), claim) {
					t.Errorf("a past exchange claims continuity with %q: %s", claim, row)
				}
			}
		}
		// no two rows in a row say the same thing
		for j := 1; j < len(vv.Settled); j++ {
			if vv.Settled[j] == vv.Settled[j-1] {
				t.Errorf("the column repeats itself at row %d: %q", j, vv.Settled[j])
			}
		}
		if i < len(said)-1 {
			if err := s.Advance(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// A visit where the visitor never moves is a real visit, and the column
// says so once rather than four times.
func TestAVisitThatNeverMovesSaysSoOnce(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	// alert bark is worth nothing to either archetype, so the band holds
	for i := 0; i < visitor.ExchangesPerScene; i++ {
		if err := s.Vocalize(AlertBark); err != nil {
			t.Fatal(err)
		}
		if i < visitor.ExchangesPerScene-1 {
			if err := s.Advance(); err != nil {
				t.Fatal(err)
			}
		}
	}
	settled := s.View(t0).Visitor.Settled
	if len(settled) != 1 {
		t.Errorf("a visit that never moved should be one row, got %d: %v", len(settled), settled)
	}
}

// The way forward is different on every beat, so no two exchanges read
// as the same screen.
func TestTheWayForwardChangesEveryExchange(t *testing.T) {
	s := newTestSession()
	s.Advance()
	s.Advance()
	seen := map[string]bool{}
	for i := 0; i < visitor.ExchangesPerScene; i++ {
		if err := s.Vocalize(Silence); err != nil {
			t.Fatal(err)
		}
		onward := s.View(t0).Visitor.Onward
		if onward == "" {
			t.Fatalf("exchange %d has no way forward", i+1)
		}
		if seen[onward] {
			t.Errorf("exchange %d repeats an earlier label: %q", i+1, onward)
		}
		seen[onward] = true
		if i < visitor.ExchangesPerScene-1 {
			if err := s.Advance(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
