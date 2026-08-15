package dogsheet

import "time"

// Every generated claim about a real dog is one of these two types,
// nothing else exists, see the verified-animal-data skill.

// VerifiedFact comes from the listing and only from the listing:
// structured fields pass through untouched, description facts are
// verbatim quotes checked against the sanitized text.
type VerifiedFact struct {
	ID          string    `json:"id"`
	Value       string    `json:"value"`
	Source      string    `json:"source"`
	RetrievedAt time.Time `json:"retrieved_at"`
}

// NarrativeInference is interpretation. It always names the facts it
// stands on and never introduces new world facts.
type NarrativeInference struct {
	Value       string   `json:"value"`
	DerivedFrom []string `json:"derived_from"`
	Category    string   `json:"category"`
}

const (
	CategoryTone     = "tone"
	CategoryBehavior = "behavior"
	CategorySensory  = "sensory"
)

// Level is the three step scale the comfort tables consume.
type Level string

const (
	LevelLow    Level = "low"
	LevelMedium Level = "medium"
	LevelHigh   Level = "high"
)

// Axis is one personality dimension. No citations means the neutral
// default, the verifier rejects uncited non medium values.
type Axis struct {
	Level       Level    `json:"level"`
	DerivedFrom []string `json:"derived_from"`
}

type Matrix struct {
	Energy           Axis `json:"energy"`
	Confidence       Axis `json:"confidence"`
	Sociability      Axis `json:"sociability"`
	Patience         Axis `json:"patience"`
	FoodDrive        Axis `json:"food_drive"`
	NoiseSensitivity Axis `json:"noise_sensitivity"`
}

// Fear may exist only when the description supports it, and it carries
// the supporting quote so the claim stays checkable forever.
type Fear struct {
	Value       string   `json:"value"`
	Quote       string   `json:"quote"`
	DerivedFrom []string `json:"derived_from"`
}

type DogSheet struct {
	AnimalID   string               `json:"animal_id"`
	Facts      []VerifiedFact       `json:"facts"`
	Matrix     Matrix               `json:"matrix"`
	Fears      []Fear               `json:"fears"`
	Quirks     []NarrativeInference `json:"quirks"`
	Movement   NarrativeInference   `json:"movement"`
	Voice      NarrativeInference   `json:"voice"`
	RadioSeed  NarrativeInference   `json:"radio_seed"`
	Default    bool                 `json:"default"`
	CompiledAt time.Time            `json:"compiled_at"`
}
