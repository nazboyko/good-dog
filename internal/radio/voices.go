package radio

import (
	"strconv"
	"strings"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
)

// Who each dog sounds like.
//
// An eleven year old six pound Maltese and a seventy eight pound hound
// must not read in the same voice, or the radio stops being a row of
// individuals and becomes one narrator doing a list. Size and age come
// off the listing, which is layer one and never guessed. The sheet's
// voice profile only moves the settings, because it is an inference and
// is not allowed to decide who somebody is.

// Voice is a library voice and how to drive it.
type Voice struct {
	ID   string
	Name string
	// Stability holds a read steady, low lets it move around
	Stability  float64
	Similarity float64
}

// Ranger hosts. He is the one invented person on the air, so he is the
// one voice not chosen from a dog's record.
var Ranger = Voice{ID: "JBFqnCBsd6RMkjVDRZzb", Name: "George", Stability: 0.5, Similarity: 0.75}

// kennelVoices is one voice per size and age. Nine of them, so twelve
// dogs cannot collapse onto two reads.
var kennelVoices = map[string]Voice{
	"small/young":   {ID: "cgSgspJ2msm6clMCkdW9", Name: "Jessica"},
	"small/adult":   {ID: "FGY2WhTYpPnrIDTdsKH5", Name: "Laura"},
	"small/senior":  {ID: "pFZP5JQG7iQjIQuC4Bku", Name: "Lily"},
	"medium/young":  {ID: "TX3LPaxmHKxFdv7VOQHJ", Name: "Liam"},
	"medium/adult":  {ID: "EXAVITQu4vr4xnSDxMaL", Name: "Sarah"},
	"medium/senior": {ID: "XrExE9yKIg1WjnnlVkGX", Name: "Matilda"},
	"large/young":   {ID: "IKne3meq5aSn9XLyUdCD", Name: "Charlie"},
	"large/adult":   {ID: "nPczCjzI2devNBz1zQrb", Name: "Brian"},
	"large/senior":  {ID: "pqHfZKP75CvOlQylNhV4", Name: "Bill"},
}

// VoiceFor picks who this dog sounds like. Pure, and the same dog
// always sounds the same, which is what makes the audio cacheable.
func VoiceFor(dog animal.Animal, sheet *dogsheet.DogSheet) Voice {
	v := kennelVoices[Size(dog)+"/"+Age(dog)]
	if v.ID == "" {
		v = kennelVoices["medium/adult"]
	}
	v.Stability, v.Similarity = settingsFor(sheet)
	return v
}

// settingsFor reads the sheet's voice profile, which is an inference
// and so may only change how a read sits, never whose read it is. A dog
// the listing calls quiet gets a steadier read, a loud one gets a read
// that moves around more.
func settingsFor(sheet *dogsheet.DogSheet) (stability, similarity float64) {
	stability, similarity = 0.5, 0.75
	if sheet == nil {
		return
	}
	profile := strings.ToLower(sheet.Voice.Value)
	switch {
	case containsAny(profile, "quiet", "calm", "soft", "gentle", "rarely"):
		stability = 0.75
	case containsAny(profile, "loud", "vocal", "barks", "excited", "energetic", "talkative"):
		stability = 0.3
	}
	return
}

// Size is small, medium or large off the listing's weight. An unlisted
// weight is medium: a guess about pitch is not a claim about a dog.
func Size(dog animal.Animal) string {
	lbs, ok := poundsIn(dog.WeightText)
	if !ok {
		return "medium"
	}
	switch {
	case lbs < 20:
		return "small"
	case lbs <= 50:
		return "medium"
	default:
		return "large"
	}
}

// Age is young, adult or senior off the listing's own age group.
func Age(dog animal.Animal) string {
	switch strings.ToLower(strings.TrimSpace(dog.AgeGroup)) {
	case "baby", "puppy", "young":
		return "young"
	case "senior":
		return "senior"
	default:
		return "adult"
	}
}

// poundsIn reads the leading number out of a weight string like
// "49 lbs". Anything it cannot read is not a weight.
func poundsIn(text string) (int, bool) {
	digits := strings.Builder{}
	for _, r := range strings.TrimSpace(text) {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
			continue
		}
		break
	}
	if digits.Len() == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(digits.String())
	if err != nil {
		return 0, false
	}
	return n, true
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
