// Package soundfx is the dog's own voice.
//
// Pressing a vocalization is the only thing the player ever does across
// three days. These are the five sounds the dog can make, recorded ahead
// of time into the same disk cache as the radio, so nothing is generated
// while somebody is playing.
//
// Silence has no sound on purpose. It is a choice the player makes, not
// a missing file, and the quiet is the point of it.
package soundfx

import (
	"fmt"

	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
	"github.com/nazboyko/good-dog/internal/session"
)

// Effect is one vocalization and the prompt that produces it.
//
// The prompts describe one animal and nothing else, because the sound
// model returns a pack of dogs in a field if the prompt does not rule it
// out. All five describe the same medium sized dog: five prompts that
// describe five different animals give a soundboard rather than a body.
//
// None of them says shelter. Barking recorded in a kennel comes back
// with the echo and the row answering, and this is the player's own
// throat, not the room. A room can be added over a dry sample later and
// it cannot be taken out of a wet one, and day three is a meeting room
// with no kennel in it. Do not improve these by adding the shelter.
//
// The adjectives stay acoustic. A growl described as a warning, or a
// whine as pleading, puts the visitor's reading inside the sound itself,
// and the gap between what the dog meant and what the human heard is the
// whole game. The sound is the middle term both lines diverge from.
type Effect struct {
	Vocalization session.Vocalization
	Prompt       string
	// Seconds is what the model is asked for, not what it returns. Short,
	// because the sound belongs to the press: it starts before the answer
	// appears and is over inside the exchange.
	Seconds float64
}

var effects = []Effect{
	{session.PlayfulBark, "A single medium sized dog gives two short happy play barks, close to the microphone, no echo, no other dogs, no music", 1.5},
	{session.AlertBark, "A single medium sized dog gives three sharp alert barks, close to the microphone, no echo, no other dogs, no music", 2},
	{session.Whine, "A single medium sized dog whines once, soft, short and high, close to the microphone, no echo, no other dogs, no music", 2},
	// two extra negatives on this one only. A grumble read as "maybe not
	// this one" teaches how little it takes; a snarl read the same way
	// tells the player the visitor was right, which is the opposite game
	{session.LowGrowl, "A single medium sized dog gives one low quiet grumbling growl deep in the chest, close to the microphone, no snarling, no barking, no echo, no other dogs, no music", 2},
	{session.Howl, "A single medium sized dog howls once, long and rising, close to the microphone, no echo, no other dogs, no music", 3},
}

// All is every sound the game can play, in the order the panel offers
// them.
func All() []Effect {
	out := make([]Effect, len(effects))
	copy(out, effects)
	return out
}

// Key is where this sound lives in the cache. It shares the radio's
// cache and its key function, so one directory holds everything the
// game plays and one prep step fills it.
//
// Every input the model is given is part of the key: the prompt, the
// duration, and the generation settings the client holds fixed. Change
// any of them and this becomes a different sound needing its own
// recording, rather than quietly keeping the old one. A key that misses
// an input reports the cache warm and plays the old file forever.
func Key(e Effect) string {
	return audiocache.Key(e.Prompt, "sfx", fmt.Sprintf("%.2fs/%s", e.Seconds, elevenlabs.SoundSettings()))
}

// Missing is every sound with no recording on disk. Boot says so out
// loud, because the client swallows a failed sound on purpose and the
// log is the only place a silent button can show up as the bug it is.
func Missing(cache *audiocache.Cache) []session.Vocalization {
	var out []session.Vocalization
	for _, e := range All() {
		if _, ok := cache.Get(Key(e)); !ok {
			out = append(out, e.Vocalization)
		}
	}
	return out
}

// Find returns the effect for a vocalization. Silence has none, which
// is the one case where a missing sound is the correct answer.
func Find(v session.Vocalization) (Effect, bool) {
	for _, e := range effects {
		if e.Vocalization == v {
			return e, true
		}
	}
	return Effect{}, false
}
