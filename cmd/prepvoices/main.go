// Command prepvoices records every radio line the game can ever play.
//
// Nothing is synthesized while somebody is listening. A night reads the
// disk cache and nothing else, so every line has to exist before the
// first player arrives. This walks the whole pool, works out every line
// each dog could say and every line the host could say about them, and
// records whatever is missing.
//
// Safe to run repeatedly: a warm cache costs nothing and generates
// nothing. Run it with `make voices`.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/config"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
	"github.com/nazboyko/good-dog/internal/httpapi"
	"github.com/nazboyko/good-dog/internal/radio"
	"github.com/nazboyko/good-dog/internal/session"
	"github.com/nazboyko/good-dog/internal/soundfx"
)

func main() {
	dry := flag.Bool("dry-run", false, "list what would be recorded and what it would cost, spend nothing")
	flag.Parse()

	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}
	key := os.Getenv("ELEVENLABS_API_KEY")
	if key == "" && !*dry {
		log.Fatal("ELEVENLABS_API_KEY not set, nothing can be recorded")
	}

	provider, err := animal.NewFixtureProvider("fixtures/dogs.json")
	if err != nil {
		log.Fatalf("open fixtures: %v", err)
	}
	compiler := dogsheet.NewCompiler(nil, dogsheet.NewCache("cache/sheets"))
	cache := audiocache.New("cache/audio")

	ctx := context.Background()
	pool, err := provider.Search(ctx)
	if err != nil {
		log.Fatalf("read the pool: %v", err)
	}

	work, err := plan(ctx, provider, compiler, pool)
	if err != nil {
		log.Fatalf("work out what to record: %v", err)
	}

	var client *elevenlabs.Client
	if key != "" {
		client = elevenlabs.New(key)
	}
	budget := &elevenlabs.Budget{}

	var toRecord []line
	for _, l := range work {
		voice := httpapi.NewHostVoice(cache, client, budget, l.Voice.ID,
			elevenlabs.VoiceSettings{Stability: l.Voice.Stability, SimilarityBoost: l.Voice.Similarity})
		if _, ok := voice.Lookup(l.Text, l.Voice); ok {
			continue
		}
		toRecord = append(toRecord, l)
	}

	chars := 0
	for _, l := range toRecord {
		chars += len(l.Text)
	}
	fmt.Printf("%d lines across %d dogs, %d already recorded, %d to record, %d characters\n",
		len(work), len(pool), len(work)-len(toRecord), len(toRecord), chars)
	byVoice(work)

	// the dog's own voice, from the sound endpoint rather than the
	// speech one. Same cache, same rule: never at play time
	every := soundfx.All()
	var sounds []soundfx.Effect
	for _, e := range every {
		if _, ok := cache.Get(soundfx.Key(e)); !ok {
			sounds = append(sounds, e)
		}
	}
	fmt.Printf("%d vocalizations, %d already recorded, %d to record\n",
		len(every), len(every)-len(sounds), len(sounds))

	if *dry {
		fmt.Println("dry run, nothing spent")
		return
	}
	if len(toRecord) == 0 && len(sounds) == 0 {
		fmt.Println("every line and every sound is already on disk, nothing to do")
		return
	}

	before := remaining(ctx, client)
	done := 0
	for _, l := range toRecord {
		voice := httpapi.NewHostVoice(cache, client, budget, l.Voice.ID,
			elevenlabs.VoiceSettings{Stability: l.Voice.Stability, SimilarityBoost: l.Voice.Similarity})
		if _, _, err := voice.Store(ctx, l.Text, l.Voice); err != nil {
			log.Printf("could not record %q as %s: %v", short(l.Text), l.Voice.Name, err)
			continue
		}
		done++
		// the library is rate limited and this is not a race
		time.Sleep(300 * time.Millisecond)
	}
	madeSounds := 0
	for _, e := range sounds {
		// sound generation is not billed per character, so this only
		// bounds a runaway loop, it does not price the call
		if err := budget.Allow(len(e.Prompt)); err != nil {
			log.Printf("budget guard stopped %s: %v", e.Vocalization, err)
			break
		}
		data, err := client.SoundEffect(ctx, e.Prompt, e.Seconds)
		if err != nil {
			log.Printf("could not record %s: %v", e.Vocalization, err)
			continue
		}
		if _, err := cache.Put(soundfx.Key(e), data); err != nil {
			log.Printf("could not cache %s: %v", e.Vocalization, err)
			continue
		}
		madeSounds++
		fmt.Printf("  %-13s %6d bytes\n", e.Vocalization, len(data))
		time.Sleep(300 * time.Millisecond)
	}
	after := remaining(ctx, client)

	fmt.Printf("recorded %d of %d lines, %d characters submitted\n", done, len(toRecord), chars)
	fmt.Printf("recorded %d of %d vocalizations\n", madeSounds, len(sounds))
	if before > 0 && after > 0 {
		fmt.Printf("credits: %d consumed, %d remaining of the month\n", before-after, after)
	}
	// a run that recorded nothing it set out to record has not succeeded,
	// and a silent cache that exits zero ships as silence nobody noticed
	if done < len(toRecord) || madeSounds < len(sounds) {
		log.Fatalf("recorded %d of %d lines and %d of %d sounds", done, len(toRecord), madeSounds, len(sounds))
	}
}

// line is one recording: what is said and who says it.
type line struct {
	Dog   string
	Text  string
	Voice radio.Voice
}

// plan works out every line the radio can ever play. A dog appears both
// as a neighbour, where the host names them, and as the dog the player
// is living as, where the whole story is theirs.
func plan(ctx context.Context, provider animal.Provider, compiler *dogsheet.Compiler, pool []animal.Animal) ([]line, error) {
	seen := map[string]bool{}
	var out []line
	add := func(dog, text string, v radio.Voice) {
		if text == "" {
			return
		}
		// Dedupe on the cache key, which carries the settings, not on
		// the voice id. Three dogs read by one library voice each carry
		// their own stability from their own sheet, so the same sentence
		// is three recordings. Deduping on the id recorded the first and
		// left the other two silent, and reported all three as done.
		key := httpapi.KeyFor(text, v)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, line{Dog: dog, Text: text, Voice: v})
	}

	// every line the host can say, across the whole rotation: any night
	// may draw any of them, so all of them have to exist
	for _, l := range radio.HostLines() {
		add("(host)", l, radio.Ranger)
	}

	for _, dog := range pool {
		org, err := provider.GetOrganization(ctx, dog.OrgID)
		if err != nil {
			continue
		}
		sheet, err := compiler.Compile(ctx, dog)
		if err != nil && sheet == nil {
			continue
		}
		voice := radio.VoiceFor(dog, sheet)
		n := radio.Neighbour{Dog: dog, Org: *org, Sheet: sheet}

		// as a neighbour: the dog says their own true thing, the host
		// names them and the place
		// the seed only picks host variants, which are already covered
		// above, so any seed enumerates the same per dog lines
		for _, c := range radio.Broadcast([]radio.Neighbour{n}, nil, voice, 0) {
			add(dog.Name, c.Line, c.Voice)
		}
		// as the dog the player is living as: the whole story is theirs
		for _, c := range radio.Broadcast(nil, session.RadioStory(dog, sheet), voice, 0) {
			add(dog.Name, c.Line, c.Voice)
		}
	}
	return out, nil
}

// byVoice prints who reads how much, which is the check that a row of
// twelve dogs did not collapse onto two voices.
func byVoice(work []line) {
	count := map[string]int{}
	dogs := map[string]map[string]bool{}
	for _, l := range work {
		count[l.Voice.Name]++
		if dogs[l.Voice.Name] == nil {
			dogs[l.Voice.Name] = map[string]bool{}
		}
		dogs[l.Voice.Name][l.Dog] = true
	}
	names := make([]string, 0, len(count))
	for n := range count {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		who := make([]string, 0, len(dogs[n]))
		for d := range dogs[n] {
			who = append(who, d)
		}
		sort.Strings(who)
		fmt.Printf("  %-9s %3d lines  %v\n", n, count[n], who)
	}
}

func remaining(ctx context.Context, c *elevenlabs.Client) int {
	if c == nil {
		return 0
	}
	sub, err := c.GetSubscription(ctx)
	if err != nil {
		return 0
	}
	return sub.Remaining()
}

func short(s string) string {
	if len(s) <= 40 {
		return s
	}
	return s[:40] + "..."
}
