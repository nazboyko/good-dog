// The player's own voice.
//
// Pressing a vocalization is the only thing the player ever does in
// three days, and it used to happen in silence. The sound belongs to the
// press, not to the answer: you hear what you did, then you read how it
// landed, which is the whole mismatch the game is about.
//
// Silence has no sound and never asks for one. It is a choice, not a
// missing file.

import type { Vocalization } from "./run";
import { classify, note } from "./speaker";

// Anything that never makes a sound. Silence is the real one; the rest
// of the list is empty on purpose, so adding a vocalization without a
// recording is a decision somebody has to make here.
const wordless: Vocalization[] = ["silence"];

// The dog only has one throat. A howl is asked for at three seconds and
// the answer lands in milliseconds, so without this the next press
// starts on top of the last one, and on the reduced motion path, where
// every transition is instant, the tail of a howl plays over the reveal.
let sounding: HTMLAudioElement | null = null;

// hush stops whatever the dog is still saying. Called on a new press and
// whenever the run leaves a screen, so nothing carries across the beat
// of dark that every transition is built around.
export function hush(): void {
  sounding?.pause?.();
  sounding = null;
}

// The sound is on top of the game and never in its way. A browser that
// blocks it, a muted tab and a judge with the sound off all land in the
// same place: the answer still lands and the screen still reads.
export function bark(
  v: Vocalization,
  make: (src: string) => HTMLAudioElement = (src) => new Audio(src),
): void {
  hush();
  if (wordless.includes(v)) return;
  try {
    // this runs inside the tap itself, so a fresh element is allowed to
    // play on iOS: the gesture is live and the element is touched in it.
    // That is the difference between this and the radio, whose cues
    // arrive seconds after any tap and have to go through primed speakers
    const sound = make(`/api/sound/${v}`);
    sounding = sound;
    const started = sound.play?.();
    // a refusal here still cannot block the press, but it is not swallowed
    // either: it is counted with the radio's, so the console says which
    // sounds a phone let through and which it did not
    if (started?.catch) started.catch((e: unknown) => note(classify(e, sound), `own voice ${v}`));
  } catch (e) {
    note("unknown", `own voice ${v}: ${String(e)}`);
  }
}
