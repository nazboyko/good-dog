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
  // the meeting room is its own place, and it arrives with ship 2 of
  // the scene layer. Until then it is the dark, which is honest enough
  // for a room the dog has never been in.
  adoption: null,
  // night has its own background and arrives with the radio
  night: null,
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

// REVEAL_BEATS is the epilogue side of the boundary, named so a test can
// assert against it rather than repeating the strings.
export const REVEAL_BEATS: readonly Beat[] = ["epilogue", "done"];
