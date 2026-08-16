package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"strings"
	"sync"
	"time"

	// header decoders for the two photo formats in the cache
	_ "image/jpeg"
	_ "image/png"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/radio"
	"github.com/nazboyko/good-dog/internal/visitor"
)

// Session is one life. It holds the real dog and the sheet, but never
// hands the client more than the current beat may know. The truth of
// where the run is lives in state, driven only by Step. The mutex
// covers state, which handlers touch from many goroutines.
type Session struct {
	ID string

	mu    sync.Mutex
	rails Rails
	state State

	dog   animal.Animal
	org   animal.Organization
	sheet *dogsheet.DogSheet
	// tonight's broadcast, built once from real neighbours. Empty is a
	// night with no radio, which the client still plays as a quiet one.
	radio []radio.Cue
}

// Tonight hands the session the broadcast for its night beat. Built
// outside because it needs the pool, which the session does not have.
func (s *Session) Tonight(cues []radio.Cue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.radio = cues
}

// View is what the client renders. Fields are filled per beat, so the
// photo path physically does not exist in the payload until epilogue.
type View struct {
	SessionID string `json:"session_id"`
	Day       int    `json:"day"`
	Beat      Beat   `json:"beat"`
	// set only once the run has ended, the reveal reads it
	Ending Ending `json:"ending,omitempty"`
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
	// how this visitor arrived, two short lines
	Arrival []string       `json:"arrival"`
	Options []Vocalization `json:"options"`
	// the visitor's own pronoun for the narrator label, never a guess
	HeardLabel string `json:"heard_label"`
	// which answer of the visit this is, and how many the visit has,
	// so the client knows whether one more is coming
	Exchange  int `json:"exchange"`
	Exchanges int `json:"exchanges"`
	// set once the player has signaled
	Signal   Vocalization `json:"signal,omitempty"`
	Mismatch *Mismatch    `json:"mismatch,omitempty"`
	// what the visitor's body says back, the only reading of comfort
	// the player ever gets
	Body string `json:"body,omitempty"`
	// the exchanges already past, oldest first, as the player can still
	// see them. The screen reads as a conversation growing rather than
	// the same frame reprinted, so it carries the visit and not only
	// its newest line. Runs of the same reading are already collapsed.
	Settled []string `json:"settled,omitempty"`
	// what the button says to leave this exchange, different on every
	// beat so the visit has a felt shape without a counter
	Onward string `json:"onward,omitempty"`
	// set only on the last exchange: the shape of the whole visit and
	// how they left
	Arc     string `json:"arc,omitempty"`
	Parting string `json:"parting,omitempty"`
}

type NightView struct {
	// the radio story about this dog, short lines, from the real sheet
	Story []string `json:"story"`
	// tonight's broadcast, the whole cue list with its offsets. The
	// stream is what plays it; this is here so a client whose stream
	// never connects can still run the night off its own timer.
	Radio []radio.Cue `json:"radio,omitempty"`
}

// EpilogueView is the reveal. It is the first and only place the photo
// and the listing url appear, and it names only verified facts. The
// moment itself gets plain words. The verbatim record sits in Listing
// and is shown only behind the quiet link, the transparency panel.
type EpilogueView struct {
	Name     string `json:"name"`
	PhotoURL string `json:"photo_url"`
	// real pixel size so the client reserves the box before the load,
	// zero when the file cannot be read and the client uses a default
	PhotoWidth  int    `json:"photo_width"`
	PhotoHeight int    `json:"photo_height"`
	ListingURL  string `json:"listing_url"`
	OrgName     string `json:"org_name"`
	// the org's name up to its first comma, for lines mid sentence:
	// "Animal Humane Society" not the full adoption center title
	OrgShort string `json:"org_short"`
	OrgCity  string `json:"org_city"`
	OrgState string `json:"org_state"`
	AgeWords string `json:"age_words,omitempty"`
	// how the three days ended, said once before the staging begins
	EndingLine string `json:"ending_line,omitempty"`
	// false only after a chosen ending, where the game has just said she
	// went home and the reveal must not also say she is waiting
	StillWaiting  bool `json:"still_waiting"`
	LongStay      bool `json:"long_stay"`
	MinutesPlayed int  `json:"minutes_played"`

	Listing ListingRecord `json:"listing"`
}

// ListingRecord is the listing as written, for the transparency panel:
// the real description next to what the game inferred from it, plus
// every listing field the game used, so the panel's promise is whole.
type ListingRecord struct {
	AgeText          string   `json:"age_text,omitempty"`
	WeightText       string   `json:"weight_text,omitempty"`
	Breed            string   `json:"breed"`
	Sex              string   `json:"sex,omitempty"`
	LongStayEvidence string   `json:"long_stay_evidence,omitempty"`
	Quotes           []string `json:"quotes"`
	Description      string   `json:"description"`
	// true when the sheet is the canned default and no quotes were read
	Default bool `json:"default"`
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func New(dog animal.Animal, org animal.Organization, sheet *dogsheet.DogSheet, rails Rails, now time.Time) *Session {
	return &Session{
		ID:    newID(),
		rails: rails,
		state: NewState(rails, now),
		dog:   dog,
		org:   org,
		sheet: sheet,
	}
}

// Resume rebuilds a session around a persisted state.
func Resume(id string, dog animal.Animal, org animal.Organization, sheet *dogsheet.DogSheet, rails Rails, state State) *Session {
	return &Session{ID: id, rails: rails, state: state, dog: dog, org: org, sheet: sheet}
}

// DogID is whose life this is. Read by the next run so living another
// life does not hand back the same animal twice in a row.
func (s *Session) DogID() string { return s.dog.ID }

// Beat is the current beat, safe to read from any goroutine.
func (s *Session) Beat() Beat {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Beat
}

// State is a copy of the truth, for persistence. Every slice in it is
// copied, so a caller can never reach the live history. That means the
// scene in progress and the signals inside each encounter as well: a
// copied struct still carries the original slice headers.
func (s *Session) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	st.Scene = append([]Vocalization{}, s.state.Scene...)
	st.Bond = append([]Encounter{}, s.state.Bond...)
	for i := range st.Bond {
		st.Bond[i].Signals = append([]Vocalization{}, st.Bond[i].Signals...)
	}
	return st
}

// tonight returns the broadcast, filtered by the same reveal guard as
// every other line on an early beat. A neighbour at the player's own
// shelter is dropped at selection, and this is the net under that.
func (s *Session) tonight() []radio.Cue {
	s.mu.Lock()
	cues := s.radio
	s.mu.Unlock()
	out := make([]radio.Cue, 0, len(cues))
	for _, c := range cues {
		if s.beforeReveal(c.Line, "") == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// apply runs one input through the pure machine under the lock.
func (s *Session) apply(in Input) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := Step(s.rails, s.state, in)
	if err != nil {
		return err
	}
	s.state = next
	return nil
}

// Advance moves to the next beat. The visitor beat refuses to advance
// until the player has signaled, the visitor is waiting for an answer.
func (s *Session) Advance() error { return s.apply(Input{Kind: InputAdvance}) }

// Vocalize records the player's signal during the visitor beat.
func (s *Session) Vocalize(v Vocalization) error {
	return s.apply(Input{Kind: InputVocalize, Signal: v})
}

// View builds the client payload for the current beat and nothing more.
func (s *Session) View(now time.Time) View {
	s.mu.Lock()
	st := s.state
	s.mu.Unlock()
	beat, signal, started := st.Beat, st.Signal, st.StartedAt

	v := View{
		SessionID: s.ID,
		Day:       st.Day,
		Beat:      beat,
		Ending:    st.Ending,
		Name:      s.dog.Name,
		AgeGroup:  s.dog.AgeGroup,
		Breed:     s.dog.Breed,
	}
	switch beat {
	case BeatScent:
		v.Scent = &ScentView{Movement: s.beforeReveal(s.sheet.Movement.Value, "")}
	case BeatVisitor, BeatAdoption:
		who := st.VisitorAtGate()
		last := beat == BeatAdoption
		past := st.pastVisits()
		vv := &VisitorView{
			Arrival:    who.Arrival,
			Options:    vocalizations,
			HeardLabel: who.Pronoun.Subject + " heard",
			Exchange:   st.Exchange(),
			Exchanges:  exchangesFor(beat),
		}
		if signal != "" {
			m := Narrate(signal)
			scene := st.SceneSoFar()
			vv.Signal = signal
			vv.Mismatch = &m
			vv.Body = visitor.Body(who, scene)
			vv.Settled = visitor.Settled(who, scene)
			vv.Onward = visitor.Onward(len(scene), exchangesFor(beat))
			if last {
				// the meeting room reads on the run's head start, so what
				// the player sees matches what is being counted
				vv.Body = visitor.AdoptionBody(who, past, scene)
				vv.Settled = visitor.AdoptionSettled(who, past, scene)
				vv.Onward = visitor.AdoptionOnward(len(scene), exchangesFor(beat))
			}
			// the visit only reads back its shape once it is over
			if len(scene) >= exchangesFor(beat) {
				end := visitor.Close(who, scene)
				if last {
					end = visitor.CloseAdoption(who, past, scene)
				}
				vv.Body = end.Body
				vv.Arc = end.Arc
				vv.Parting = end.Parting
			}
		}
		v.Visitor = vv
	case BeatNight:
		var story []string
		for _, line := range RadioStory(s.dog, s.sheet) {
			if kept := s.beforeReveal(line, ""); kept != "" {
				story = append(story, kept)
			}
		}
		v.Night = &NightView{Story: story, Radio: s.tonight()}
	case BeatEpilogue, BeatDone:
		v.Epilogue = s.epilogue(now, started, st.Ending)
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

func (s *Session) epilogue(now, started time.Time, ending Ending) *EpilogueView {
	// never nil: a default sheet has no quotes and the panel must not break
	quotes := []string{}
	for _, f := range s.sheet.Facts {
		if f.Source == "description" {
			quotes = append(quotes, f.Value)
		}
	}
	w, h := photoSize(s.dog.PhotoLocal)
	return &EpilogueView{
		Name: s.dog.Name,
		// session scoped so the route can refuse before the epilogue
		PhotoURL:      "/api/session/" + s.ID + "/photo",
		PhotoWidth:    w,
		PhotoHeight:   h,
		ListingURL:    s.dog.ListingURL,
		OrgName:       s.org.Name,
		OrgShort:      strings.TrimSpace(strings.SplitN(s.org.Name, ",", 2)[0]),
		OrgCity:       s.org.City,
		OrgState:      StateName(s.org.State),
		AgeWords:      AgeInWords(s.dog.AgeText),
		EndingLine:    EndingLine(ending, s.dog.Name, visitor.Adopter(nil).Pronoun.Object),
		StillWaiting:  StillWaiting(ending),
		LongStay:      s.dog.LongStay,
		MinutesPlayed: int(now.Sub(started).Minutes()),
		Listing: ListingRecord{
			AgeText:          s.dog.AgeText,
			WeightText:       s.dog.WeightText,
			Breed:            s.dog.Breed,
			Sex:              s.dog.Sex,
			LongStayEvidence: s.dog.LongStayEvidence,
			Quotes:           quotes,
			Description:      s.dog.Description,
			Default:          s.sheet.Default,
		},
	}
}

// photoSize reads only the image header, no decode, so the reveal can
// reserve the photo's box before a single byte of it is shown.
func photoSize(path string) (width, height int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// StateName spells out a US postal code for the reveal, and passes
// anything it does not know through untouched.
func StateName(code string) string {
	if name, ok := stateNames[strings.ToUpper(code)]; ok {
		return name
	}
	return code
}

var stateNames = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas", "CA": "California",
	"CO": "Colorado", "CT": "Connecticut", "DE": "Delaware", "FL": "Florida", "GA": "Georgia",
	"HI": "Hawaii", "ID": "Idaho", "IL": "Illinois", "IN": "Indiana", "IA": "Iowa",
	"KS": "Kansas", "KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi", "MO": "Missouri",
	"MT": "Montana", "NE": "Nebraska", "NV": "Nevada", "NH": "New Hampshire", "NJ": "New Jersey",
	"NM": "New Mexico", "NY": "New York", "NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio",
	"OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah", "VT": "Vermont",
	"VA": "Virginia", "WA": "Washington", "WV": "West Virginia", "WI": "Wisconsin", "WY": "Wyoming",
	"DC": "Washington, DC",
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

// Store is the in memory cache over the durable DB. Every Put snapshots
// to the DB, every Get miss falls through to the DB and rebuilds the
// session, so a restart loses nothing.
type Store struct {
	mu       sync.Mutex
	sessions map[string]*Session
	db       *DB
	// rebuild turns a persisted row back into a live session: the dog,
	// the org and the sheet come from the provider and the sheet cache
	rebuild func(ctx context.Context, row Row) (*Session, error)
}

// NewStore takes the DB and a rebuild func. A nil DB is a memory only
// store, which the tests and a scratch server use.
func NewStore(db *DB, rebuild func(ctx context.Context, row Row) (*Session, error)) *Store {
	return &Store{sessions: map[string]*Session{}, db: db, rebuild: rebuild}
}

// Put caches the session and snapshots it. Called on create and after
// every transition, so the DB always holds the latest truth.
func (st *Store) Put(ctx context.Context, s *Session, now time.Time) error {
	st.mu.Lock()
	st.sessions[s.ID] = s
	st.mu.Unlock()
	if st.db == nil {
		return nil
	}
	return st.db.Save(ctx, Row{
		ID: s.ID, DogID: s.dog.ID, OrgID: s.org.ID, Rails: s.rails.Name(),
		State: s.State(), UpdatedAt: now,
	})
}

// Get returns the cached session, or rebuilds it from the DB. A miss
// anywhere is ErrNotFound, and the caller starts a clean run.
func (st *Store) Get(ctx context.Context, id string) (*Session, error) {
	st.mu.Lock()
	s, ok := st.sessions[id]
	st.mu.Unlock()
	if ok {
		return s, nil
	}
	if st.db == nil || st.rebuild == nil {
		return nil, ErrNotFound
	}
	row, err := st.db.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	s, err = st.rebuild(ctx, row)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	st.mu.Lock()
	// another request may have rebuilt it first, keep the first one
	if cached, ok := st.sessions[id]; ok {
		s = cached
	} else {
		st.sessions[id] = s
	}
	st.mu.Unlock()
	return s, nil
}

// forget drops a session from the cache only, so a test can simulate
// a restart without closing the DB.
func (st *Store) forget(id string) {
	st.mu.Lock()
	delete(st.sessions, id)
	st.mu.Unlock()
}
