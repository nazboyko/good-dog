import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { listen, STREAM_GRACE_MS, type EventSourceLike, type RadioCue } from "./broadcast";

const cues: RadioCue[] = [
  { at_ms: 100, speaker: "ranger", line: "Shelter radio, and it is late." },
  { at_ms: 300, speaker: "story", line: "That is Keno, over at Animal Humane Society in Golden Valley." },
  { at_ms: 500, speaker: "ranger", line: "Sleep if you can." },
];

// a stand in for EventSource that lets a test decide what the network does
function fakeSource() {
  const listeners = new Map<string, ((e: Event) => void)[]>();
  let closed = false;
  const source: EventSourceLike = {
    addEventListener: (type, fn) => {
      listeners.set(type, [...(listeners.get(type) ?? []), fn]);
    },
    close: () => {
      closed = true;
    },
  };
  return {
    source,
    closed: () => closed,
    emit(type: string, data?: unknown) {
      for (const fn of listeners.get(type) ?? []) {
        fn({ data: JSON.stringify(data ?? {}) } as MessageEvent);
      }
    },
  };
}

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

test("the stream drives the night when it is alive", () => {
  const played: number[] = [];
  let done = false;
  let fellBack = false;
  const fake = fakeSource();
  listen("abc", cues, {
    onCue: (i) => played.push(i),
    onDone: () => (done = true),
    onFallback: () => (fellBack = true),
  }, () => fake.source);

  fake.emit("hello");
  fake.emit("radio", { index: 0 });
  fake.emit("radio", { index: 1 });
  fake.emit("radio", { index: 2 });
  fake.emit("radio_done");

  expect(played).toEqual([0, 1, 2]);
  expect(done).toBe(true);
  expect(fellBack).toBe(false);

  // and the local timers never fire on top of it
  vi.advanceTimersByTime(5000);
  expect(played).toEqual([0, 1, 2]);
});

// The night has to happen even with the stream dead. A judge on a bad
// connection should see a night, not a blank screen.
test("a stream that never speaks hands the night to a local timer", () => {
  const played: number[] = [];
  let done = false;
  let fellBack = false;
  const fake = fakeSource();
  listen("abc", cues, {
    onCue: (i) => played.push(i),
    onDone: () => (done = true),
    onFallback: () => (fellBack = true),
  }, () => fake.source);

  // silence: no hello, no cues
  vi.advanceTimersByTime(STREAM_GRACE_MS);
  expect(fellBack).toBe(true);
  expect(fake.closed()).toBe(true);
  expect(played).toEqual([]);

  // the whole night still plays, on the cue list's own offsets
  vi.advanceTimersByTime(100);
  expect(played).toEqual([0]);
  vi.advanceTimersByTime(200);
  expect(played).toEqual([0, 1]);
  vi.advanceTimersByTime(200);
  expect(played).toEqual([0, 1, 2]);
  expect(done).toBe(false);
  vi.advanceTimersByTime(400);
  expect(done).toBe(true);
});

test("a stream that errors before it speaks also falls back", () => {
  const played: number[] = [];
  const fake = fakeSource();
  listen("abc", cues, { onCue: (i) => played.push(i), onDone: () => {} }, () => fake.source);

  fake.emit("error");
  vi.advanceTimersByTime(600);
  expect(played).toEqual([0, 1, 2]);
});

// A stream that spoke and then dropped is EventSource's problem: it
// reconnects on its own, and a local timer racing it would double up.
test("the fallback never runs twice or races a live stream", () => {
  const played: number[] = [];
  const fake = fakeSource();
  listen("abc", cues, { onCue: (i) => played.push(i), onDone: () => {} }, () => fake.source);

  fake.emit("hello");
  fake.emit("radio", { index: 0 });
  fake.emit("error");
  fake.emit("error");
  vi.advanceTimersByTime(5000);

  // the fallback replays the night once, and nothing plays a cue twice
  const counts = new Map<number, number>();
  for (const i of played) counts.set(i, (counts.get(i) ?? 0) + 1);
  expect([...counts.values()].every((n) => n === 1)).toBe(true);
});

test("stopping the night cancels the stream and every pending cue", () => {
  const played: number[] = [];
  const fake = fakeSource();
  const stop = listen("abc", cues, { onCue: (i) => played.push(i), onDone: () => {} }, () => fake.source);

  stop();
  expect(fake.closed()).toBe(true);
  vi.advanceTimersByTime(10000);
  expect(played).toEqual([]);
});

test("a night with no cues still settles instead of hanging", () => {
  let done = false;
  const fake = fakeSource();
  listen("abc", [], { onCue: () => {}, onDone: () => (done = true) }, () => fake.source);
  vi.advanceTimersByTime(STREAM_GRACE_MS + 500);
  expect(done).toBe(true);
});
