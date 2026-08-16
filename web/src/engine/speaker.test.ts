import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { classify, note, prime, reset, take, tally } from "./speaker";
import { play, type RadioCue } from "./broadcast";
import { bark } from "./voice";

// A stand in for an audio element that remembers whether it was touched
// in the gesture and lets a test choose what play() does later.
function speakerEl(refuse?: string) {
  const el = {
    src: "",
    preload: "",
    error: null as { code: number } | null,
    touched: 0,
    loaded: 0,
    listeners: {} as Record<string, (() => void)[]>,
    setAttribute() {},
    load() {
      this.loaded++;
    },
    addEventListener(t: string, fn: () => void) {
      (this.listeners[t] ??= []).push(fn);
    },
    play() {
      this.touched++;
      if (refuse) {
        const e = new Error(refuse);
        e.name = refuse;
        return Promise.reject(e);
      }
      return Promise.resolve();
    },
    end() {
      (this.listeners.ended ?? []).forEach((f) => f());
    },
  };
  return el as unknown as HTMLAudioElement & typeof el;
}

beforeEach(() => {
  reset();
  vi.useFakeTimers();
  vi.spyOn(console, "warn").mockImplementation(() => {});
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

// The rule iOS enforces: the element that plays a cue has to be one that
// was touched inside a gesture. So a cue must play through a primed
// speaker, not through a fresh element made when the cue arrived.
test("a radio cue plays through a speaker that was touched in the tap", async () => {
  const made: ReturnType<typeof speakerEl>[] = [];
  prime(() => {
    const el = speakerEl();
    made.push(el);
    return el;
  });
  expect(made.length).toBeGreaterThan(0);
  for (const el of made) expect(el.touched).toBe(1); // touched in the gesture

  const cue: RadioCue = { at_ms: 0, speaker: "story", line: "x", audio: "/api/audio/a.mp3" };
  const p = play(cue);
  const used = made.find((el) => el.src === "/api/audio/a.mp3");
  expect(used, "the cue did not go through a primed speaker").toBeTruthy();
  expect(used!.touched).toBe(2); // primed once, then played for real
  used!.end();
  await expect(p).resolves.toBe("finished");
  expect(tally().played).toBe(1);
  expect(tally().primed).toBe(true);
});

test("priming twice does not grow the pool, and every speaker is touched again", () => {
  const made: ReturnType<typeof speakerEl>[] = [];
  const make = () => {
    const el = speakerEl();
    made.push(el);
    return el;
  };
  prime(make);
  const n = made.length;
  prime(make);
  expect(made.length).toBe(n);
  for (const el of made) expect(el.touched).toBe(2);
  expect(tally().pool).toBe(n);
});

// A refusal is never silent. It is counted under the reason the browser
// gave, so someone with the console open can tell iOS said no from the
// file never having arrived.
test("iOS refusing a line is counted as not_allowed and said out loud", async () => {
  prime(() => speakerEl("NotAllowedError"));
  const cue: RadioCue = { at_ms: 0, speaker: "story", line: "x", audio: "/api/audio/a.mp3" };
  await expect(play(cue)).resolves.toBe("blocked");
  expect(tally().not_allowed).toBe(1);
  expect(tally().played).toBe(0);
  expect(console.warn).toHaveBeenCalledWith(expect.stringContaining("not_allowed"));
});

test("a file that never arrived is counted as no_source, not as a refusal", async () => {
  prime(() => {
    const el = speakerEl();
    // the element reports the media error, then fires error
    el.play = () => {
      (el as { error: unknown }).error = { code: 4 };
      queueMicrotask(() => (el.listeners.error ?? []).forEach((f) => f()));
      return Promise.resolve();
    };
    return el;
  });
  const cue: RadioCue = { at_ms: 0, speaker: "story", line: "x", audio: "/api/audio/gone.mp3" };
  await expect(play(cue)).resolves.toBe("blocked");
  expect(tally().no_source).toBe(1);
  expect(tally().not_allowed).toBe(0);
});

test("classify reads the browser's own names", () => {
  expect(classify({ name: "NotAllowedError" })).toBe("not_allowed");
  expect(classify({ name: "AbortError" })).toBe("aborted");
  expect(classify(null, { error: { code: 4 } } as unknown as HTMLAudioElement)).toBe("no_source");
  expect(classify(null, { error: { code: 3 } } as unknown as HTMLAudioElement)).toBe("decode");
  expect(classify(new Error("what"))).toBe("unknown");
});

// The dog's own voice plays inside the tap, so it uses a fresh element
// and iOS allows it. It still has to be counted if it is not.
test("the dog's own voice counts a refusal too, and never throws", () => {
  expect(() => bark("howl", () => speakerEl("NotAllowedError"))).not.toThrow();
  return vi.waitFor(() => expect(tally().not_allowed).toBe(1));
});

// Without a prime, which is the desktop first-load path and every test
// that never tapped, a cue still plays through a fresh element.
test("with nothing primed a cue still gets an element", async () => {
  const el = speakerEl();
  const cue: RadioCue = { at_ms: 0, speaker: "story", line: "x", audio: "/api/audio/a.mp3" };
  const p = play(cue, () => {
    el.src = "/api/audio/a.mp3";
    return el;
  });
  el.end();
  await expect(p).resolves.toBe("finished");
  expect(take(() => el)).toBe(el);
  expect(tally().primed).toBe(false);
});

test("note counts and warns, played counts quietly", () => {
  note("played");
  note("not_allowed", "x");
  expect(tally().played).toBe(1);
  expect(tally().not_allowed).toBe(1);
  expect(console.warn).toHaveBeenCalledTimes(1);
});
