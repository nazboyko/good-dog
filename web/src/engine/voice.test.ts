import { expect, test } from "vitest";
import { bark, hush } from "./voice";
import { VOCALIZATION_LABELS, type Vocalization } from "./run";

// Ground truth is the game's own list of vocalizations, so a new one
// added without a sound shows up here rather than as a silent button.
const every = Object.keys(VOCALIZATION_LABELS) as Vocalization[];

test("every vocalization but silence reaches for its own sound", () => {
  const asked: string[] = [];
  for (const v of every) {
    bark(v, (src) => { asked.push(src); return { play: () => Promise.resolve() } as unknown as HTMLAudioElement; });
  }
  expect(asked).toEqual([
    "/api/sound/playful_bark",
    "/api/sound/alert_bark",
    "/api/sound/whine",
    "/api/sound/low_growl",
    "/api/sound/howl",
  ]);
  // and no two presses share a sound
  expect(new Set(asked).size).toBe(asked.length);
});

test("silence is silent on purpose", () => {
  const asked: string[] = [];
  bark("silence", (src) => { asked.push(src); return {} as HTMLAudioElement; });
  expect(asked).toEqual([]);
});

// The sound sits on top of the game and can never be in its way.
test("a press still counts when the sound cannot play", () => {
  expect(() => bark("howl", () => ({ play: () => Promise.reject(new Error("NotAllowedError")) }) as unknown as HTMLAudioElement)).not.toThrow();
  expect(() => bark("howl", () => { throw new Error("no audio here"); })).not.toThrow();
  expect(() => bark("howl", () => ({}) as HTMLAudioElement)).not.toThrow();
});

// The dog has one throat. A howl is asked for at three seconds and the
// answer lands in milliseconds, so without this the next press starts on
// top of the last, and on the reduced motion path the tail of a howl
// plays over the reveal.
test("a new press stops whatever the dog was still saying", () => {
  const paused: string[] = [];
  const speaker = (name: string) =>
    ({ play: () => Promise.resolve(), pause: () => paused.push(name) }) as unknown as HTMLAudioElement;

  bark("howl", () => speaker("howl"));
  expect(paused).toEqual([]);
  bark("whine", () => speaker("whine"));
  expect(paused).toEqual(["howl"]);
});

test("leaving the screen stops the sound", () => {
  const paused: string[] = [];
  bark("howl", () => ({ play: () => Promise.resolve(), pause: () => paused.push("howl") }) as unknown as HTMLAudioElement);
  hush();
  expect(paused).toEqual(["howl"]);
  // and a second hush has nothing left to stop
  hush();
  expect(paused).toEqual(["howl"]);
});
