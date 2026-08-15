package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
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

func newTestSession() *Session {
	return New(testDog, testOrg, testSheet(), t0)
}

func TestRailsRunInOrder(t *testing.T) {
	s := newTestSession()
	want := []Beat{BeatWake, BeatScent, BeatVisitor, BeatNight, BeatEpilogue, BeatDone}
	for i, beat := range want {
		if s.Beat() != beat {
			t.Fatalf("step %d: beat = %s, want %s", i, s.Beat(), beat)
		}
		if beat == BeatVisitor {
			if err := s.Vocalize(Whine); err != nil {
				t.Fatal(err)
			}
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
	s := New(testDog, testOrg, sheet, t0)

	s.Advance()
	if got := s.View(t0).Scent.Movement; got != "" {
		t.Errorf("movement naming the city must be dropped before the reveal, got %q", got)
	}
	s.Advance()
	s.Vocalize(Silence)
	s.Advance()
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

func TestStoreRoundTrip(t *testing.T) {
	st := NewStore()
	s := newTestSession()
	st.Put(s)
	got, ok := st.Get(s.ID)
	if !ok || got != s {
		t.Fatal("store must return the same session")
	}
	if _, ok := st.Get("nope"); ok {
		t.Fatal("unknown id must miss")
	}
}
