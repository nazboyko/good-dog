package radio

import (
	"context"
	"fmt"
)

// The spoken half of the radio.
//
// Nothing is ever synthesized while somebody is listening. The host says
// the same three lines every night, so the whole spoken set is prepared
// once, before any night can start, and every night after that reads it
// off the disk cache. A miss at play time is a bug and is reported as
// one: it is never quietly covered by generating the line right then,
// because that would put a paid network call with no budget ceiling in
// the middle of a beat that is supposed to be paced to the second.
//
// The text is the night. The voice is on top of it. A line with no
// recording still plays as a line.

// VoiceCache is the disk cache and the generator behind it. Kept as an
// interface so this package stays free of the audio vendor and can be
// tested without one.
type VoiceCache interface {
	// Lookup finds an already prepared recording. No network, no cost.
	Lookup(text string) (file string, ok bool)
	// Store synthesizes and caches one line, returning the file and the
	// characters it cost. Only ever called from Prepare.
	Store(ctx context.Context, text string) (file string, chars int, err error)
}

// Prepared is what one preparation run did, for the log and for the
// operator who wants to know what it cost.
type Prepared struct {
	Lines     int
	AlreadyIn int
	Generated int
	Chars     int
	Files     map[string]string
}

func (p Prepared) String() string {
	return fmt.Sprintf("host voice ready: %d lines, %d already cached, %d generated, %d characters",
		p.Lines, p.AlreadyIn, p.Generated, p.Chars)
}

// Prepare makes sure every host line has a recording on disk. Safe to
// run on every boot: a warm cache costs nothing and generates nothing.
//
// An error here does not stop the game. A silent radio is a worse night
// than a spoken one and a much better night than no night, so the caller
// logs and carries on with whatever was prepared.
func Prepare(ctx context.Context, vc VoiceCache) (Prepared, error) {
	out := Prepared{Files: map[string]string{}}
	for _, line := range HostLines() {
		out.Lines++
		if file, ok := vc.Lookup(line); ok {
			out.AlreadyIn++
			out.Files[line] = file
			continue
		}
		file, chars, err := vc.Store(ctx, line)
		if err != nil {
			return out, fmt.Errorf("preparing %q: %w", line, err)
		}
		out.Generated++
		out.Chars += chars
		out.Files[line] = file
	}
	return out, nil
}

// Missing lists host lines with no recording on disk. Empty is the only
// healthy answer once Prepare has run, and the server says so loudly at
// boot rather than letting a player find out at midnight.
func Missing(vc VoiceCache) []string {
	var out []string
	for _, line := range HostLines() {
		if _, ok := vc.Lookup(line); !ok {
			out = append(out, line)
		}
	}
	return out
}

// WithVoices attaches recordings to the cues that have one. It only ever
// reads the cache. A line whose recording is missing keeps its text and
// loses its audio, and the caller is told which ones so a miss shows up
// as the bug it is instead of as silence nobody notices.
func WithVoices(cues []Cue, vc VoiceCache) ([]Cue, []string) {
	if vc == nil {
		return cues, nil
	}
	var missing []string
	out := make([]Cue, len(cues))
	copy(out, cues)
	for i, c := range out {
		if c.Speaker != SpeakerRanger {
			continue
		}
		file, ok := vc.Lookup(c.Line)
		if !ok {
			missing = append(missing, c.Line)
			continue
		}
		out[i].Audio = file
	}
	return out, missing
}
