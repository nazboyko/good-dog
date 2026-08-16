package radio

import "context"

// The spoken half of the radio.
//
// Nothing is ever synthesized while somebody is listening. The host
// draws from a small fixed rotation and every dog's lines are fixed
// too, so the whole spoken set is recorded once by `make voices` and
// every night after that reads it off the disk cache. A miss at play
// time is a bug and is reported as one: it is never quietly covered by
// generating the line right then, because that would put a paid network
// call with no budget ceiling in the middle of a beat paced to the
// second.
//
// The text is the night. The voice is on top of it. A line with no
// recording still plays as a line.

// VoiceCache is the disk cache and the generator behind it. Kept as an
// interface so this package stays free of the audio vendor and can be
// tested without one.
type VoiceCache interface {
	// Lookup finds an already prepared recording. No network, no cost.
	// Keyed by the voice as well as the text: the same sentence read by
	// the host and by a dog are two different recordings.
	Lookup(text string, v Voice) (file string, ok bool)
	// Store synthesizes and caches one line, returning the file and the
	// characters it cost. Only ever called from preparation, never from
	// a night in progress.
	Store(ctx context.Context, text string, v Voice) (file string, chars int, err error)
}

// Missing lists host lines with no recording on disk.
//
// The server checks this at boot and says so loudly, and that is all it
// does: it never records. Recording is `make voices` and nothing else,
// because a boot that quietly synthesizes is a boot that quietly spends
// money, and adding four variants to a rotation should not cost
// anything until somebody asks for it.
func Missing(vc VoiceCache) []string {
	var out []string
	for _, line := range HostLines() {
		if _, ok := vc.Lookup(line, Ranger); !ok {
			out = append(out, line)
		}
	}
	return out
}

// WithVoices attaches recordings to every cue that has one, each in the
// voice that cue was written for. It only ever reads the cache. A line
// whose recording is missing keeps its text and loses its audio, and
// the caller is told which ones so a miss shows up as the bug it is
// instead of as silence nobody notices.
func WithVoices(cues []Cue, vc VoiceCache) ([]Cue, []string) {
	if vc == nil {
		return cues, nil
	}
	var missing []string
	out := make([]Cue, len(cues))
	copy(out, cues)
	for i, c := range out {
		file, ok := vc.Lookup(c.Line, c.Voice)
		if !ok {
			missing = append(missing, c.Line)
			continue
		}
		out[i].Audio = file
	}
	return out, missing
}
