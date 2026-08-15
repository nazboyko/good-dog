package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
)

// Session is one life. It holds the real dog and the sheet, but never
// hands the client more than the current beat may know. The mutex
// covers beat and signal, which handlers touch from many goroutines.
type Session struct {
	ID        string
	StartedAt time.Time

	mu     sync.Mutex
	beat   Beat
	signal Vocalization

	dog   animal.Animal
	org   animal.Organization
	sheet *dogsheet.DogSheet
}

// View is what the client renders. Fields are filled per beat, so the
// photo path physically does not exist in the payload until epilogue.
type View struct {
	SessionID string `json:"session_id"`
	Beat      Beat   `json:"beat"`
	// the player knows only name, age group and breed at the start
	Name     string `json:"name"`
	AgeGroup string `json:"age_group"`
	Breed    string `json:"breed"`

	Scent    *ScentView    `json:"scent,omitempty"`
	Visitor  *VisitorView  `json:"visitor,omitempty"`
	Night    *NightView    `json:"night,omitempty"`
	Epilogue *EpilogueView `json:"epilogue,omitempty"`
}

type ScentView struct {
	// one grounded line about how this dog moves through the room
	Movement string `json:"movement"`
}

type VisitorView struct {
	Options []Vocalization `json:"options"`
	// set once the player has signaled
	Signal   Vocalization `json:"signal,omitempty"`
	Mismatch *Mismatch    `json:"mismatch,omitempty"`
}

type NightView struct {
	// the radio story, short lines with pauses, from the real sheet
	Story []string `json:"story"`
}

// EpilogueView is the reveal. It is the first and only place the photo
// and the listing url appear, and it names only verified facts.
type EpilogueView struct {
	Name          string   `json:"name"`
	PhotoURL      string   `json:"photo_url"`
	ListingURL    string   `json:"listing_url"`
	OrgName       string   `json:"org_name"`
	OrgCity       string   `json:"org_city"`
	OrgState      string   `json:"org_state"`
	AgeText       string   `json:"age_text,omitempty"`
	WeightText    string   `json:"weight_text,omitempty"`
	LongStay      bool     `json:"long_stay"`
	MinutesPlayed int      `json:"minutes_played"`
	Quotes        []string `json:"quotes"`
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func New(dog animal.Animal, org animal.Organization, sheet *dogsheet.DogSheet, now time.Time) *Session {
	return &Session{
		ID:        newID(),
		StartedAt: now,
		beat:      BeatWake,
		dog:       dog,
		org:       org,
		sheet:     sheet,
	}
}

// Beat is the current beat, safe to read from any goroutine.
func (s *Session) Beat() Beat {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.beat
}

// Advance moves to the next beat. The visitor beat refuses to advance
// until the player has signaled, the visitor is waiting for an answer.
func (s *Session) Advance() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.beat == BeatVisitor && s.signal == "" {
		return fmt.Errorf("the visitor is waiting, choose a signal first")
	}
	if s.beat == BeatDone {
		return fmt.Errorf("this life is over")
	}
	s.beat = next(s.beat)
	return nil
}

// Vocalize records the player's signal during the visitor beat.
func (s *Session) Vocalize(v Vocalization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.beat != BeatVisitor {
		return fmt.Errorf("nobody is here to hear that right now")
	}
	if !ValidVocalization(v) {
		return fmt.Errorf("unknown vocalization %q", v)
	}
	s.signal = v
	return nil
}

// View builds the client payload for the current beat and nothing more.
func (s *Session) View(now time.Time) View {
	s.mu.Lock()
	beat, signal := s.beat, s.signal
	s.mu.Unlock()

	v := View{
		SessionID: s.ID,
		Beat:      beat,
		Name:      s.dog.Name,
		AgeGroup:  s.dog.AgeGroup,
		Breed:     s.dog.Breed,
	}
	switch beat {
	case BeatScent:
		v.Scent = &ScentView{Movement: s.beforeReveal(s.sheet.Movement.Value, "")}
	case BeatVisitor:
		vv := &VisitorView{Options: vocalizations}
		if signal != "" {
			m := Narrate(signal)
			vv.Signal = signal
			vv.Mismatch = &m
		}
		v.Visitor = vv
	case BeatNight:
		var story []string
		for _, line := range RadioStory(s.dog, s.sheet) {
			if kept := s.beforeReveal(line, ""); kept != "" {
				story = append(story, kept)
			}
		}
		v.Night = &NightView{Story: story}
	case BeatEpilogue, BeatDone:
		v.Epilogue = s.epilogue(now)
	}
	return v
}

// beforeReveal guards generated text on the early beats: a sheet line
// that names the shelter or its city would spoil the reveal, so it is
// replaced with the fallback, which may be empty to drop the line.
func (s *Session) beforeReveal(line, fallback string) string {
	lower := strings.ToLower(line)
	for _, secret := range []string{s.org.Name, s.org.City} {
		if secret != "" && strings.Contains(lower, strings.ToLower(secret)) {
			return fallback
		}
	}
	return line
}

func (s *Session) epilogue(now time.Time) *EpilogueView {
	var quotes []string
	for _, f := range s.sheet.Facts {
		if f.Source == "description" {
			quotes = append(quotes, f.Value)
		}
		if len(quotes) == 3 {
			break
		}
	}
	return &EpilogueView{
		Name: s.dog.Name,
		// session scoped so the route can refuse before the epilogue
		PhotoURL:      "/api/session/" + s.ID + "/photo",
		ListingURL:    s.dog.ListingURL,
		OrgName:       s.org.Name,
		OrgCity:       s.org.City,
		OrgState:      s.org.State,
		AgeText:       s.dog.AgeText,
		WeightText:    s.dog.WeightText,
		LongStay:      s.dog.LongStay,
		MinutesPlayed: int(now.Sub(s.StartedAt).Minutes()),
		Quotes:        quotes,
	}
}

// PhotoLocal is exposed only for the photo route, which itself refuses
// to serve before the epilogue beat.
func (s *Session) PhotoLocal() string { return s.dog.PhotoLocal }

// RadioStory turns a real sheet into Old Ranger's short lines. Every
// line stands on a verified fact or the sheet's cited radio seed. It
// ends with the real name, per the shelter-radio skill. The place is
// held back on purpose: this is the player's own dog, and the shelter
// is part of the reveal, so it waits for the epilogue.
func RadioStory(dog animal.Animal, sheet *dogsheet.DogSheet) []string {
	p := pronounsFor(dog.Sex)
	lines := []string{
		"This one is for the dog in the third kennel down.",
		sheet.RadioSeed.Value,
	}
	if len(sheet.Quirks) > 0 {
		lines = append(lines, strings.TrimSpace(sheet.Quirks[0].Value))
	}
	if dog.LongStay {
		// the fact is the filing, never how long, so the wait stays vague
		lines = append(lines, fmt.Sprintf("%s %s been waiting a while.", p.subject, p.have))
	}
	lines = append(lines,
		fmt.Sprintf("%s name is %s.", p.possessive, dog.Name),
		fmt.Sprintf("%s %s real, and %s %s still here.", p.subject, p.is, strings.ToLower(p.subject), p.is),
	)
	return lines
}

type pronounSet struct {
	subject, possessive, is, have string
}

// pronounsFor reads the listing's sex field, the only verified source.
// Unknown or unlisted falls back to they, never a guess from the name.
func pronounsFor(sex string) pronounSet {
	switch strings.ToLower(sex) {
	case "female":
		return pronounSet{"She", "Her", "is", "has"}
	case "male":
		return pronounSet{"He", "His", "is", "has"}
	}
	return pronounSet{"They", "Their", "are", "have"}
}

// Store keeps live sessions in memory. The prototype does not survive a
// restart yet, that is the sqlite step on the plan.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func NewStore() *Store {
	return &Store{sessions: map[string]*Session{}}
}

func (st *Store) Put(s *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sessions[s.ID] = s
}

func (st *Store) Get(id string) (*Session, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, ok := st.sessions[id]
	return s, ok
}
