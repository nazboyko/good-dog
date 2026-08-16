package radio

import (
	"testing"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/dogsheet"
)

func dog(weight, ageGroup string) animal.Animal {
	return animal.Animal{WeightText: weight, AgeGroup: ageGroup}
}

// The whole point of this unit, in one test. If a six pound senior and a
// seventy eight pound adult read in the same voice, the row stops being
// individuals and becomes one narrator working through a list.
func TestASmallSeniorAndALargeAdultDoNotSoundAlike(t *testing.T) {
	maltese := VoiceFor(dog("6 lbs", "Senior"), nil)
	hound := VoiceFor(dog("78 lbs", "Adult"), nil)
	if maltese.ID == hound.ID {
		t.Errorf("both read as %s, the row has one voice", maltese.Name)
	}
	if maltese.ID == Ranger.ID || hound.ID == Ranger.ID {
		t.Error("a dog must not read in the host's voice")
	}
}

// Nine buckets, nine voices, no accidental duplicates: a table where two
// buckets share an id quietly halves the row.
func TestEveryBucketHasItsOwnVoice(t *testing.T) {
	seen := map[string]string{}
	for bucket, v := range kennelVoices {
		if v.ID == "" {
			t.Errorf("%s has no voice", bucket)
		}
		if other, dup := seen[v.ID]; dup {
			t.Errorf("%s and %s are the same voice %s", other, bucket, v.Name)
		}
		seen[v.ID] = bucket
		if v.ID == Ranger.ID {
			t.Errorf("%s is the host's voice", bucket)
		}
	}
	if len(kennelVoices) != 9 {
		t.Errorf("three sizes by three ages is nine buckets, got %d", len(kennelVoices))
	}
}

func TestSizeAndAgeComeOffTheListing(t *testing.T) {
	cases := []struct{ weight, age, wantSize, wantAge string }{
		{"6 lbs", "Senior", "small", "senior"},
		{"19 lbs", "Adult", "small", "adult"},
		{"20 lbs", "Adult", "medium", "adult"},
		{"50 lbs", "Baby", "medium", "young"},
		{"51 lbs", "Young", "large", "young"},
		{"78 lbs", "Adult", "large", "adult"},
		// an unlisted weight is not a guess about pitch
		{"", "Adult", "medium", "adult"},
		{"unknown", "Adult", "medium", "adult"},
	}
	for _, c := range cases {
		d := dog(c.weight, c.age)
		if got := Size(d); got != c.wantSize {
			t.Errorf("weight %q is %s, want %s", c.weight, got, c.wantSize)
		}
		if got := Age(d); got != c.wantAge {
			t.Errorf("age group %q is %s, want %s", c.age, got, c.wantAge)
		}
	}
}

// The voice profile is an inference, so it may change how a read sits
// and never whose read it is. Layer two does not get to pick a body.
func TestTheVoiceProfileMovesSettingsNotIdentity(t *testing.T) {
	d := dog("40 lbs", "Adult")
	quiet := VoiceFor(d, &dogsheet.DogSheet{Voice: dogsheet.NarrativeInference{Value: "quiet until something matters"}})
	loud := VoiceFor(d, &dogsheet.DogSheet{Voice: dogsheet.NarrativeInference{Value: "vocal and excited at the gate"}})
	if quiet.ID != loud.ID {
		t.Error("the sheet changed who the dog is, it may only change how they read")
	}
	if quiet.Stability <= loud.Stability {
		t.Errorf("a quiet dog should read steadier: quiet %v loud %v", quiet.Stability, loud.Stability)
	}
	// and a dog with no sheet still has a voice
	if VoiceFor(d, nil).ID == "" {
		t.Error("a dog with no sheet still has to sound like something")
	}
}

// Same dog, same voice, every time: the disk cache depends on it.
func TestVoiceChoiceIsStable(t *testing.T) {
	d := dog("6 lbs", "Senior")
	sheet := &dogsheet.DogSheet{Voice: dogsheet.NarrativeInference{Value: "quiet"}}
	first := VoiceFor(d, sheet)
	for i := 0; i < 20; i++ {
		if VoiceFor(d, sheet) != first {
			t.Fatal("the same dog read differently twice, nothing would ever cache")
		}
	}
}

// The pool is twelve real dogs and the row must not collapse onto two
// reads. This walks the sizes and ages the fixtures actually contain.
func TestTheRealPoolSpreadsAcrossVoices(t *testing.T) {
	// the shapes in fixtures/dogs.json, by weight and age group
	pool := []animal.Animal{
		dog("6 lbs", "Senior"), dog("11 lbs", "Adult"), dog("13 lbs", "Baby"),
		dog("40 lbs", "Adult"), dog("49 lbs", "Adult"), dog("55 lbs", "Adult"),
		dog("78 lbs", "Adult"), dog("60 lbs", "Senior"), dog("25 lbs", "Young"),
	}
	seen := map[string]bool{}
	for _, d := range pool {
		seen[VoiceFor(d, nil).Name] = true
	}
	if len(seen) < 5 {
		t.Errorf("nine shapes of dog produced only %d voices: %v", len(seen), seen)
	}
	// the user's own example, which is the whole point of the unit
	maltese := VoiceFor(dog("6 lbs", "Senior"), nil)
	hound := VoiceFor(dog("78 lbs", "Adult"), nil)
	if maltese.Name == hound.Name {
		t.Errorf("the six pound senior and the seventy eight pound hound both read as %s", maltese.Name)
	}
}
