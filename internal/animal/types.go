package animal

import "time"

// Status is the whole lifecycle the game is allowed to know about.
// The word adopted is only ever true with ADOPTED_CONFIRMED.
type Status string

const (
	StatusActive           Status = "ACTIVE"
	StatusRemovedUnknown   Status = "REMOVED_UNKNOWN"
	StatusTransferred      Status = "TRANSFERRED"
	StatusUnavailable      Status = "UNAVAILABLE"
	StatusAdoptedConfirmed Status = "ADOPTED_CONFIRMED"
)

func ValidStatus(s Status) bool {
	switch s {
	case StatusActive, StatusRemovedUnknown, StatusTransferred, StatusUnavailable, StatusAdoptedConfirmed:
		return true
	}
	return false
}

// Animal is the normalized internal shape. Provider payloads stop at the
// adapter, the rest of the game only ever sees this.
type Animal struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Breed       string    `json:"breed"`
	AgeGroup    string    `json:"age_group"`
	Sex         string    `json:"sex"`
	Size        string    `json:"size"`
	Description string    `json:"description"`
	PhotoURL    string    `json:"photo_url"`
	PhotoLocal  string    `json:"photo_local"`
	ListingURL  string    `json:"listing_url"`
	OrgID       string    `json:"org_id"`
	Status      Status    `json:"status"`
	RetrievedAt time.Time `json:"retrieved_at"`
	// Synthetic marks shape examples, they must never reach a player
	Synthetic bool `json:"synthetic,omitempty"`
}

type Organization struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
	URL   string `json:"url"`
}
