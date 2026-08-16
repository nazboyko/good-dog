import type { Beat } from "./run";

// Where the dog is, per beat, and the hard edge where dog vision stops.
//
// The whole run is seen through a dog's eyes: dichromatic blue and
// yellow, no red, no green. That is what makes the epilogue land. The
// reveal is the first true image in the game, the real photo in the
// real colors of a real animal, and it only reads that way because
// everything before it did not. So the shader is not "off" at the
// epilogue, it ends there, and this table is where that is decided.
//
// Record<Beat, ...> is the enforcement: a new beat is a compile error
// here until somebody decides which side of the boundary it is on.
// Only the beats the dog is living through have a scene. A beat that
// resolves to null renders no canvas at all, which is stronger than
// rendering an unshaded one: there is nothing on the page to shade.
const SCENES: Record<Beat, string | null> = {
  wake: "/scenes/enclosure.webp",
  scent: "/scenes/enclosure.webp",
  visitor: "/scenes/enclosure.webp",
  // The meeting room is its own place, and the room is the point of day
  // three: she has never been in it, and it does not have a gate.
  adoption: "/scenes/dating-room.webp",
  // the same row after lights out, which is where the radio is
  night: "/scenes/night-enclosure.webp",
  // the boundary. never give these a scene.
  epilogue: null,
  done: null,
};

// sceneFor is the only place a beat becomes a background. Anything that
// wants to draw asks here first, so the boundary cannot be forgotten at
// a call site.
export function sceneFor(beat: Beat): string | null {
  return SCENES[beat] ?? null;
}

// How long a room takes to replace another. Matches --fade-quiet, and
// the old room is held on screen for exactly this long after the new one
// paints, so the two overlap the whole way through.
export const CROSSFADE_MS = 520;

// How much ground goes over each room so the words stay readable.
//
// Measured, not guessed. Each number is the one where muted text over
// the brightest part of the middle band, at the 99.9th percentile, is at
// least as good as the enclosure the copy was originally tuned against,
// which sits at 3.34 to 1.
//
// The night row is the counterintuitive one. It reads far darker, and
// the instinct is to lift the veil, but that is the mean talking: the
// light down the corridor is as bright as anything in the daytime room,
// and at 0.55 muted text over it falls to 1.8. It keeps the same veil as
// the day and is still plainly night, because its mean is less than half
// the day room's and the veil does not change that.
//
// The meeting room is the opposite and behaves as expected: much lighter
// throughout, so it needs more before the floor comes back.
const VEILS: Record<string, number> = {
  // 3.34 muted, the number every other room has to match
  "/scenes/enclosure.webp": 0.8,
  // 3.78 muted. At 0.55 this was 1.83, which is unreadable
  "/scenes/night-enclosure.webp": 0.8,
  // 4.06 muted. At 0.8 it was 3.29, just under the floor
  "/scenes/dating-room.webp": 0.86,
};

export function veilFor(scene: string): number {
  return VEILS[scene] ?? 0.8;
}

// REVEAL_BEATS is the epilogue side of the boundary, named so a test can
// assert against it rather than repeating the strings.
export const REVEAL_BEATS: readonly Beat[] = ["epilogue", "done"];
