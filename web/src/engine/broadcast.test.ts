import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { listen, play, queue, visibleCues, BEAT_MS, STALLED_MS, STREAM_GRACE_MS, type EventSourceLike, type RadioCue } from "./broadcast";

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
    onAllSent: () => (done = true),
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
    onAllSent: () => (done = true),
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
  listen("abc", cues, { onCue: (i) => played.push(i), onAllSent: () => {} }, () => fake.source);

  fake.emit("error");
  vi.advanceTimersByTime(600);
  expect(played).toEqual([0, 1, 2]);
});

// A stream that spoke and then dropped is EventSource's problem: it
// reconnects on its own, and a local timer racing it would double up.
test("the fallback never runs twice or races a live stream", () => {
  const played: number[] = [];
  const fake = fakeSource();
  listen("abc", cues, { onCue: (i) => played.push(i), onAllSent: () => {} }, () => fake.source);

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
  const stop = listen("abc", cues, { onCue: (i) => played.push(i), onAllSent: () => {} }, () => fake.source);

  stop();
  expect(fake.closed()).toBe(true);
  vi.advanceTimersByTime(10000);
  expect(played).toEqual([]);
});

test("a night with no cues still settles instead of hanging", () => {
  let done = false;
  const fake = fakeSource();
  listen("abc", [], { onCue: () => {}, onAllSent: () => (done = true) }, () => fake.source);
  vi.advanceTimersByTime(STREAM_GRACE_MS + 500);
  expect(done).toBe(true);
});

// Sound is on top of the night, never the night itself. Some judges
// will watch with the sound off, and every one of these paths has to
// land in the same place: the line is on screen and the night goes on.
test("audio never blocks a line, and says which way it went", async () => {
  const made: string[] = [];
  const withAudio: RadioCue = { at_ms: 0, speaker: "ranger", line: "Shelter radio.", audio: "/api/audio/x.mp3" };
  const silent: RadioCue = { at_ms: 0, speaker: "story", line: "That is Bella." };

  // an element that can be told to reach its end
  const speaker = () => {
    const ends: (() => void)[] = [];
    const errors: (() => void)[] = [];
    const el = {
      addEventListener: (type: string, fn: () => void) => {
        if (type === "ended") ends.push(fn);
        if (type === "error") errors.push(fn);
      },
      play: () => Promise.resolve(),
    } as unknown as HTMLAudioElement;
    return { el, end: () => ends.forEach((f) => f()), fail: () => errors.forEach((f) => f()) };
  };

  // a cue with no recording asks for nothing and is blocked at once
  await expect(play(silent, (src) => { made.push(src); return {} as HTMLAudioElement; })).resolves.toBe("blocked");
  expect(made).toEqual([]);

  // a cue with one plays it, and stays pending until the sound is over.
  // Resolving when playback merely starts is the overlap bug itself, so
  // the promise has to be caught still waiting before ended fires.
  const good = speaker();
  let settled = false;
  const finished = play(withAudio, (src) => { made.push(src); return good.el; });
  void finished.then(() => (settled = true));
  expect(made).toEqual(["/api/audio/x.mp3"]);
  await vi.advanceTimersByTimeAsync(5000);
  expect(settled).toBe(false);
  good.end();
  await expect(finished).resolves.toBe("finished");

  // the file is there but will not decode
  const broken = speaker();
  const bad = play(withAudio, () => broken.el);
  broken.fail();
  await expect(bad).resolves.toBe("blocked");

  // autoplay refused
  await expect(
    play(withAudio, () => ({ addEventListener: () => {}, play: () => Promise.reject(new Error("NotAllowedError")) }) as unknown as HTMLAudioElement),
  ).resolves.toBe("blocked");

  // no Audio at all, or a constructor that throws
  await expect(play(withAudio, () => { throw new Error("no audio here"); })).resolves.toBe("blocked");

  // a recording that never ends and never errors cannot hold the night
  const stalled = play(withAudio, () => ({ addEventListener: () => {}, play: () => Promise.resolve() }) as unknown as HTMLAudioElement);
  await vi.advanceTimersByTimeAsync(STALLED_MS + 10);
  await expect(stalled).resolves.toBe("finished");
});

// The whole night still runs when every recording is missing, which is
// the same night a player with the sound off gets.
test("a night with no recordings plays every line", () => {
  const silentCues: RadioCue[] = cues.map((c) => ({ ...c, audio: undefined }));
  const played: number[] = [];
  let done = false;
  const fake = fakeSource();
  listen("abc", silentCues, { onCue: (i) => played.push(i), onAllSent: () => (done = true) }, () => fake.source);
  fake.emit("hello");
  fake.emit("radio", { index: 0 });
  fake.emit("radio", { index: 1 });
  fake.emit("radio", { index: 2 });
  fake.emit("radio_done");
  expect(played).toEqual([0, 1, 2]);
  expect(done).toBe(true);
});

// Sixteen lines accumulating turns a broadcast into a transcript. A
// short window rolls instead, so the night is heard rather than read.
test("the night holds a moving window, not a document", () => {
  const night: RadioCue[] = [
    { at_ms: 0, speaker: "ranger", line: "Shelter radio, and it is late." },
    { at_ms: 1, speaker: "ranger", line: "Lights are down." },
    { at_ms: 2, speaker: "story", line: "Keno one." },
    { at_ms: 3, speaker: "story", line: "That is Keno." },
    { at_ms: 4, speaker: "story", line: "Bella one." },
    { at_ms: 5, speaker: "story", line: "That is Bella." },
    { at_ms: 6, speaker: "own", line: "Her name is Venus." },
    { at_ms: 7, speaker: "own", line: "She is real, and she is still here." },
    { at_ms: 8, speaker: "ranger", line: "Sleep if you can." },
  ];

  // nothing heard, nothing shown
  expect(visibleCues(night, 0)).toEqual([]);
  // early on, everything heard so far fits inside the window
  expect(visibleCues(night, 2)).toEqual([0, 1]);
  expect(visibleCues(night, 3)).toEqual([0, 1, 2]);
  // then the oldest neighbour lines roll off
  expect(visibleCues(night, 4)).toEqual([1, 2, 3]);
  expect(visibleCues(night, 6)).toEqual([3, 4, 5]);

  // the player's own dog stays, and the neighbours keep rolling past it
  expect(visibleCues(night, 7)).toEqual([4, 5, 6]);
  expect(visibleCues(night, 8)).toEqual([5, 6, 7]);

  // the whole own segment is still there at the end, with the closer
  const atEnd = visibleCues(night, night.length);
  expect(atEnd).toContain(6);
  expect(atEnd).toContain(7);
  expect(atEnd).toContain(8);
  // and it has not become the transcript again
  expect(atEnd.length).toBeLessThan(night.length);
});

// Every line is still a subtitle when it is on screen, and no line is
// ever skipped: the window moves, it does not drop lines unheard.
test("the window shows a prefix of what has been heard and never skips", () => {
  const night: RadioCue[] = Array.from({ length: 16 }, (_, i) => ({
    at_ms: i,
    speaker: i >= 12 && i < 15 ? "own" : "story",
    line: `line ${i}`,
  }));
  const everShown = new Set<number>();
  for (let heard = 1; heard <= night.length; heard++) {
    const visible = visibleCues(night, heard);
    visible.forEach((i) => everShown.add(i));
    // never shows a line that has not been heard
    expect(Math.max(...visible)).toBeLessThan(heard);
    // always shows the newest one
    expect(visible).toContain(heard - 1);
  }
  // every line in the night was on screen at some point
  expect(everShown.size).toBe(night.length);
});

// ---- the queue: one voice at a time ----

// A radio that keeps its own books. It records when each line started
// and stopped, so a test can ask whether two voices were ever live at
// the same moment without asking the queue what it thinks it did.
function fakeRadio(durations: Record<number, number>) {
  const live = new Set<string>();
  const overlaps: string[][] = [];
  const order: string[] = [];
  return {
    overlaps,
    order,
    playCue(cue: RadioCue): Promise<"finished" | "blocked"> {
      const id = cue.line;
      if (live.size > 0) overlaps.push([...live, id]);
      live.add(id);
      order.push(id);
      return new Promise((resolve) => {
        setTimeout(() => {
          live.delete(id);
          resolve("finished");
        }, durations[Number(cue.at_ms)] ?? 1000);
      });
    },
  };
}

// The offsets the server really sends on, from radio.Broadcast: a
// couple of seconds between lines, tuned for reading them.
const voiced: RadioCue[] = [
  { at_ms: 1200, speaker: "ranger", line: "one", audio: "/a/1.mp3" },
  { at_ms: 3800, speaker: "story", line: "two", audio: "/a/2.mp3" },
  { at_ms: 6200, speaker: "story", line: "three", audio: "/a/3.mp3" },
];

test("no two voices are ever live at the same moment", async () => {
  const radio = fakeRadio({ 1200: 1000, 3800: 1000, 6200: 1000 });
  let done = false;
  const q = queue(voiced, { onCue: () => {}, onDone: () => (done = true) }, { playCue: radio.playCue });

  // the server hands all three over on its own schedule, well before
  // any of them has finished speaking
  q.arrive(0);
  q.arrive(1);
  q.arrive(2);
  q.noMore();

  await vi.advanceTimersByTimeAsync(10000);

  expect(radio.overlaps).toEqual([]);
  expect(radio.order).toEqual(["one", "two", "three"]);
  expect(done).toBe(true);
});

test("a line waits for the one before it to finish, plus a beat", async () => {
  const at: number[] = [];
  let now = 0;
  const q = queue(
    voiced,
    { onCue: () => at.push(now), onDone: () => {} },
    {
      playCue: () => new Promise((r) => setTimeout(() => r("finished"), 1000)),
      wait: (ms) => new Promise((r) => setTimeout(r, ms)),
    },
  );
  const tick = setInterval(() => (now += 10), 10);
  q.arrive(0);
  q.arrive(1);
  q.arrive(2);
  await vi.advanceTimersByTimeAsync(6000);
  clearInterval(tick);

  // first line goes up at once, each one after it a full recording plus
  // a beat later, never on the 200ms slots the server sent them on
  expect(at[0]).toBe(0);
  expect(at[1]).toBeGreaterThanOrEqual(1000 + BEAT_MS);
  expect(at[2]).toBeGreaterThanOrEqual(2 * (1000 + BEAT_MS));
});

// The muted and audio blocked nights are the nights they always were.
test("an unvoiced night keeps the server's reading pace exactly", async () => {
  const silent: RadioCue[] = [
    { at_ms: 0, speaker: "ranger", line: "one" },
    { at_ms: 1200, speaker: "story", line: "two" },
    { at_ms: 3800, speaker: "story", line: "three" },
  ];
  const at: number[] = [];
  let now = 0;
  const q = queue(silent, { onCue: () => at.push(now), onDone: () => {} }, {
    wait: (ms) => new Promise((r) => setTimeout(r, ms)),
  });
  const tick = setInterval(() => (now += 10), 10);
  q.arrive(0);
  q.arrive(1);
  q.arrive(2);
  await vi.advanceTimersByTimeAsync(10000);
  clearInterval(tick);

  // ground truth is the schedule the server wrote, not the queue's idea
  // of it: each line sits for the distance to the next cue
  expect(at[0]).toBe(0);
  expect(at[1]).toBe(1200);
  expect(at[2]).toBe(3800);
});

test("a blocked recording falls back to reading pace, not to a beat", async () => {
  const at: number[] = [];
  let now = 0;
  const q = queue(
    voiced,
    { onCue: () => at.push(now), onDone: () => {} },
    {
      playCue: () => Promise.resolve("blocked"),
      wait: (ms) => new Promise((r) => setTimeout(r, ms)),
    },
  );
  const tick = setInterval(() => (now += 10), 10);
  q.arrive(0);
  q.arrive(1);
  await vi.advanceTimersByTimeAsync(4000);
  clearInterval(tick);

  // the 2600ms slot the server wrote, not the 700ms beat
  expect(at[1]).toBe(2600);
});

// Sleep unlocks when the night is over, which is later than the moment
// the server finished handing the cues across.
test("the night is not over until the last voice has finished", async () => {
  let done = false;
  const q = queue(
    voiced,
    { onCue: () => {}, onDone: () => (done = true) },
    { playCue: () => new Promise((r) => setTimeout(() => r("finished"), 1000)) },
  );
  q.arrive(0);
  q.arrive(1);
  q.arrive(2);
  q.noMore();

  // the server has said everything it is going to say
  await vi.advanceTimersByTimeAsync(1500);
  expect(done).toBe(false);

  await vi.advanceTimersByTimeAsync(10000);
  expect(done).toBe(true);
});

// A reconnect makes the server replay the night from cue zero.
test("a replayed cue is never heard twice", async () => {
  const radio = fakeRadio({ 1200: 100, 3800: 100, 6200: 100 });
  const q = queue(voiced, { onCue: () => {}, onDone: () => {} }, { playCue: radio.playCue });
  q.arrive(0);
  q.arrive(0);
  q.arrive(1);
  q.arrive(0);
  q.arrive(2);
  await vi.advanceTimersByTimeAsync(10000);
  expect(radio.order).toEqual(["one", "two", "three"]);
});

test("stopping the queue leaves the rest of the night unplayed", async () => {
  const radio = fakeRadio({ 1200: 100, 3800: 100, 6200: 100 });
  let done = false;
  const q = queue(voiced, { onCue: () => {}, onDone: () => (done = true) }, { playCue: radio.playCue });
  q.arrive(0);
  q.arrive(1);
  q.arrive(2);
  q.noMore();
  await vi.advanceTimersByTimeAsync(150);
  q.stop();
  await vi.advanceTimersByTimeAsync(10000);
  expect(radio.order.length).toBeLessThan(3);
  expect(done).toBe(false);
});

// The night the game actually plays: cues arrive seconds apart and some
// recordings are shorter than the slot they were written for, so the
// queue runs dry between lines and has to wake up for each arrival.
// Handing every cue over at once, the way the tests above do, never
// exercises that and lets a queue that only ever drains once look fine.
test("the queue wakes up again for a cue that arrives after it ran dry", async () => {
  const radio = fakeRadio({ 1200: 300, 3800: 300, 6200: 300 });
  let done = false;
  const q = queue(voiced, { onCue: () => {}, onDone: () => (done = true) }, { playCue: radio.playCue });

  q.arrive(0);
  await vi.advanceTimersByTimeAsync(2600);
  // the first line is long finished and nothing is left waiting
  expect(radio.order).toEqual(["one"]);
  expect(done).toBe(false);

  q.arrive(1);
  await vi.advanceTimersByTimeAsync(2400);
  expect(radio.order).toEqual(["one", "two"]);
  expect(done).toBe(false);

  q.arrive(2);
  q.noMore();
  await vi.advanceTimersByTimeAsync(2000);
  expect(radio.order).toEqual(["one", "two", "three"]);
  expect(radio.overlaps).toEqual([]);
  expect(done).toBe(true);
});

// Sleep must stay locked through every gap in a night that arrives
// piecemeal, not just through one unbroken drain.
test("sleep stays locked while the server is still sending", async () => {
  const radio = fakeRadio({ 1200: 300, 3800: 300, 6200: 300 });
  let done = false;
  const q = queue(voiced, { onCue: () => {}, onDone: () => (done = true) }, { playCue: radio.playCue });

  for (const i of [0, 1, 2]) {
    q.arrive(i);
    await vi.advanceTimersByTimeAsync(3000);
    // every line so far has been heard and the queue is empty, but the
    // server has not said it is finished, so the night is not over
    expect(done).toBe(false);
  }
  q.noMore();
  await vi.advanceTimersByTimeAsync(50);
  expect(done).toBe(true);
});

// The last line of a muted night still gets a moment before sleep opens.
test("the closing line of an unvoiced night is not cut short", async () => {
  const silent: RadioCue[] = [
    { at_ms: 0, speaker: "ranger", line: "one" },
    { at_ms: 1200, speaker: "ranger", line: "two" },
  ];
  let done = false;
  const q = queue(silent, { onCue: () => {}, onDone: () => (done = true) }, {});
  q.arrive(0);
  q.arrive(1);
  q.noMore();
  await vi.advanceTimersByTimeAsync(1200 + BEAT_MS - 50);
  expect(done).toBe(false);
  await vi.advanceTimersByTimeAsync(100);
  expect(done).toBe(true);
});

// A reconnect can replay the whole night, radio_done included.
test("a second radio_done cannot end the night twice", async () => {
  let ends = 0;
  const q = queue(voiced, { onCue: () => {}, onDone: () => ends++ }, { playCue: () => Promise.resolve("finished") });
  q.arrive(0);
  q.arrive(1);
  q.arrive(2);
  q.noMore();
  await vi.advanceTimersByTimeAsync(5000);
  q.noMore();
  await vi.advanceTimersByTimeAsync(5000);
  expect(ends).toBe(1);
});
