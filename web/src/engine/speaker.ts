// The speakers the radio plays through, and why they exist.
//
// On iOS an audio element may only start playing if that same element
// was touched inside a user gesture. Not the page: the element. Desktop
// browsers unlock the whole page on the first tap, which is why the
// night played on a laptop and stayed silent on a phone. Every radio cue
// used to make a fresh Audio element seconds after the tap, from a
// stream event, and iOS refused each one.
//
// So the elements are made and touched here, inside the tap that starts
// the night, before any cue exists. A pool of blank speakers, each one
// played once on nothing while the gesture is still live. Later, when a
// cue arrives, it takes a speaker, sets its src and plays, and iOS
// treats that as the continuation of the tap. That is the rule iOS
// actually enforces and the only way through it.
//
// Nothing about the muted night changes: if a speaker still cannot play,
// the cue reports blocked and the line stays on screen for its reading
// time, exactly as before. What changes is that the refusal is now
// counted and said out loud, so a silent night is never a mystery.

// How many lines can be in flight at once. The queue plays one at a
// time, but a speaker is not free until its sound has ended, and the
// next line starts a beat after that, so two covers the handover with
// one to spare for a stalled element.
const POOL = 3;

let pool: HTMLAudioElement[] = [];
let primedInGesture = false;

// What happened when the pipeline was last asked to make a sound. Read
// by the console diagnostics and by the failure count, so someone with
// the phone in their hand can tell "iOS refused" from "the file never
// came" without guessing.
export type Refusal = "not_allowed" | "no_source" | "decode" | "aborted" | "unknown";

const counts: Record<Refusal | "played", number> = {
  played: 0,
  not_allowed: 0,
  no_source: 0,
  decode: 0,
  aborted: 0,
  unknown: 0,
};

export function tally(): Readonly<typeof counts> & { primed: boolean; pool: number } {
  return { ...counts, primed: primedInGesture, pool: pool.length };
}

// prime is the one call that has to happen inside a real user gesture.
// It builds the speakers and touches each one so iOS marks it as
// activated. Safe to call again: it only builds what is missing and
// touches what is there. Cheap, silent, and it never throws.
export function prime(make: () => HTMLAudioElement = () => new Audio()): void {
  try {
    while (pool.length < POOL) {
      const el = make();
      // playsinline keeps iOS from opening a full screen player for a
      // radio line, and preload none keeps the blank touch from fetching
      el.setAttribute("playsinline", "");
      el.preload = "auto";
      pool.push(el);
    }
    for (const el of pool) {
      // play on nothing: iOS records the gesture on the element and then
      // fails the play for lack of a source, which is fine and expected
      const p = el.play?.();
      if (p?.catch) p.catch(() => {});
      // some iOS versions want a load() in the gesture as well
      try {
        el.load?.();
      } catch {
        // an element that cannot load here will report itself later
      }
    }
    primedInGesture = true;
  } catch {
    // no Audio at all: the night is text, and tally() says primed false
  }
}

// take hands out a speaker for one line. Rotates through the pool so a
// speaker that is still finishing is not reused underneath its own
// sound. Falls back to a fresh element when nothing was primed, which is
// the desktop path and the path a test takes.
let next = 0;
export function take(make: () => HTMLAudioElement = () => new Audio()): HTMLAudioElement {
  if (pool.length === 0) return make();
  const el = pool[next % pool.length];
  next++;
  return el;
}

// classify turns whatever the element or its play() promise said into
// one of the reasons that matter, so the count means something.
export function classify(err: unknown, el?: HTMLAudioElement | null): Refusal {
  const name = (err as { name?: string } | undefined)?.name ?? "";
  if (name === "NotAllowedError") return "not_allowed";
  if (name === "AbortError") return "aborted";
  const code = el?.error?.code;
  // MediaError: 1 aborted, 2 network, 3 decode, 4 src not supported
  if (code === 4 || code === 2) return "no_source";
  if (code === 3) return "decode";
  if (code === 1) return "aborted";
  if (name === "NotSupportedError") return "no_source";
  return "unknown";
}

export function note(how: Refusal | "played", detail?: string): void {
  counts[how]++;
  if (how === "played") return;
  // Loud on purpose. A radio line that made no sound is a bug or a
  // policy, and either way somebody with the console open must be able
  // to tell which without a debugger.
  console.warn(`radio: line did not play (${how}${detail ? ": " + detail : ""}); primed in gesture: ${primedInGesture}`);
}

// for tests, and for a judge typing it into the console
export function reset(): void {
  pool = [];
  primedInGesture = false;
  next = 0;
  for (const k of Object.keys(counts) as (keyof typeof counts)[]) counts[k] = 0;
}
