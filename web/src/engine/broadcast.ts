// The night radio, client side.
//
// Two clocks, and they are not the same one. The server decides when a
// cue is handed over, on offsets tuned for reading: that schedule lives
// over there because a backgrounded tab throttles timers. The queue here
// decides when a cue is heard, and a spoken line takes as long as it
// takes. A line that has no voice falls back to the server's reading
// pace, so a night with the sound off is the night it always was.
//
// The stream is the radio. listen() is the plan for when there is no
// radio, and queue() is what stops three dogs talking at once.

export type RadioCue = {
  at_ms: number;
  speaker: "ranger" | "story" | "own";
  line: string;
  // the recording of this line, absent when there is none
  audio?: string;
};

// What became of a line's recording. "finished" means it played to the
// end and the next line can follow it. "blocked" means no sound reached
// the room, whether that was the autoplay policy, a missing file or a
// browser with no Audio at all, and the night falls back to reading pace.
export type Played = "finished" | "blocked";

// The silence between two speakers on the radio. Long enough to hear the
// handover, short enough that the night does not sag.
export const BEAT_MS = 700;

// Nothing on this radio is half a minute long. If a recording stalls
// without ever ending or erroring, the night stops waiting on it: one
// silent gap is survivable, a night that never reaches sleep is not.
export const STALLED_MS = 30000;

// play starts a cue's recording and resolves when the room is quiet
// again. It never rejects and it never reports an error upward, because
// the subtitle is the night and the voice is on top of it: a browser
// that refuses to autoplay, a missing file and a judge watching with the
// sound off all have to land in the same place, which is a night that
// reads.
export function play(
  cue: RadioCue,
  make: (src: string) => HTMLAudioElement = (src) => new Audio(src),
  // hands the element back so a night that is being torn down can stop
  // the sound it already started
  onSound?: (sound: HTMLAudioElement) => void,
): Promise<Played> {
  if (!cue.audio) return Promise.resolve("blocked");
  return new Promise<Played>((resolve) => {
    let settled = false;
    let stall: ReturnType<typeof setTimeout>;
    const done = (how: Played) => {
      if (settled) return;
      settled = true;
      // without this the element stays reachable from a pending timer
      // for half a minute after it has finished, once per line
      clearTimeout(stall);
      resolve(how);
    };
    let sound: HTMLAudioElement;
    try {
      sound = make(cue.audio!);
    } catch {
      done("blocked");
      return;
    }
    onSound?.(sound);
    sound.addEventListener?.("ended", () => done("finished"));
    sound.addEventListener?.("error", () => done("blocked"));
    // a rejection here is the autoplay policy, not an error
    const started = sound.play?.();
    if (started?.catch) started.catch(() => done("blocked"));
    stall = setTimeout(() => done("finished"), STALLED_MS);
  });
}

// How many of the neighbours' lines stay on screen at once.
//
// Sixteen lines accumulating turns a broadcast into a transcript: the
// player stops listening and starts reading, and the night becomes a
// document to get through. A short window rolls instead, so the screen
// holds what is being said rather than everything that has been said.
export const WINDOW = 3;

// visibleCues is which lines are on screen once heard of them have
// arrived, oldest first.
//
// The player's own dog and the last line of the night are never rolled
// away. The night ends on them, and an ending you have to scroll back
// for is not an ending. Everything else is a window.
export function visibleCues(cues: RadioCue[], heard: number): number[] {
  const out: number[] = [];
  for (let i = 0; i < Math.min(heard, cues.length); i++) {
    const isLast = i === cues.length - 1;
    const stays = cues[i].speaker === "own" || isLast;
    if (stays || i >= heard - WINDOW) out.push(i);
  }
  return out;
}

// readingGap is how long a line with no voice stays up: the distance to
// the next cue on the server's schedule, which is the pace the whole
// night ran at before any of it was recorded. Keeping it means the
// muted and audio blocked nights are the nights they always were.
function readingGap(cues: RadioCue[], index: number): number {
  const next = cues[index + 1];
  if (!next) return BEAT_MS;
  return Math.max(0, next.at_ms - cues[index].at_ms);
}

export type QueueHandlers = {
  // put the line on screen. Called before its sound starts, so the night
  // reads the same whether or not the sound arrives
  onCue: (index: number) => void;
  // the last line has finished and the room is quiet
  onDone: () => void;
};

export type Queue = {
  // a cue is ready to be heard. Out of order and repeat arrivals are
  // dropped, so a reconnect that replays the night cannot double it
  arrive: (index: number) => void;
  // nothing more is coming. The night ends once the queue drains
  noMore: () => void;
  stop: () => void;
};

// queue plays the night one line at a time.
//
// The offsets the server sends on are tuned for reading, and a spoken
// line runs longer than its slot, so pacing playback by them puts three
// dogs on the air at once. Arrivals buffer here instead and each line
// waits for the one before it to finish. The night gets as long as it
// needs to be.
export function queue(
  cues: RadioCue[],
  handlers: QueueHandlers,
  deps: {
    playCue?: (cue: RadioCue) => Promise<Played>;
    wait?: (ms: number) => Promise<void>;
  } = {},
): Queue {
  let sounding: HTMLAudioElement | null = null;
  const playCue = deps.playCue ?? ((cue: RadioCue) => play(cue, undefined, (s) => (sounding = s)));
  const wait = deps.wait ?? ((ms: number) => new Promise<void>((r) => setTimeout(r, ms)));

  const waiting: number[] = [];
  let highest = -1;
  let draining = false;
  let sealed = false;
  let stopped = false;
  let finished = false;

  const drain = async () => {
    if (draining) return;
    draining = true;
    while (waiting.length > 0 && !stopped) {
      const index = waiting.shift()!;
      handlers.onCue(index);
      // the sound has to be over before the next line starts, which is
      // the whole point: only one radio line is ever audible
      const how = await playCue(cues[index]);
      await wait(how === "finished" ? BEAT_MS : readingGap(cues, index));
    }
    draining = false;
    // sealed is what keeps sleep locked: the night is over when the last
    // line has been heard, not when the server stopped sending
    if (sealed && !stopped && !finished) {
      finished = true;
      handlers.onDone();
    }
  };

  return {
    arrive(index) {
      if (stopped || index <= highest) return;
      highest = index;
      waiting.push(index);
      void drain();
    },
    noMore() {
      sealed = true;
      void drain();
    },
    stop() {
      stopped = true;
      // two nights running at once is what StrictMode's second mount
      // would otherwise give, and only one of them can be silenced by
      // dropping the queue
      sounding?.pause?.();
      sounding = null;
    },
  };
}

// How long to wait for the stream to say anything before deciding it is
// not going to. The server sends hello the instant it accepts the
// connection, so silence past this is a real failure, not slowness.
export const STREAM_GRACE_MS = 2500;

export type BroadcastHandlers = {
  onCue: (index: number) => void;
  // every cue has been handed over. Not the end of the night: the last
  // one still has to be heard, and that is the queue's to decide
  onAllSent: () => void;
  // called when the stream never spoke and the night ran on its own
  onFallback?: () => void;
};

// listen opens the stream for one session's night and returns a stop
// function. If the stream has not said hello within the grace period,
// or it errors before the first cue, the night plays from the cue list
// on a local timer instead. A judge on a flaky connection sees a night,
// not a blank screen.
export function listen(
  sessionId: string,
  cues: RadioCue[],
  handlers: BroadcastHandlers,
  open: (url: string) => EventSourceLike = (url) => new EventSource(url),
): () => void {
  let stopped = false;
  let usingFallback = false;
  // the highest cue index already played. A cue never plays twice, which
  // covers both a fallback starting after the stream got some of the way
  // through and an EventSource reconnect, which makes the server open a
  // fresh broadcast from cue zero.
  let highest = -1;
  const timers: ReturnType<typeof setTimeout>[] = [];
  let source: EventSourceLike | null = null;

  const clearTimers = () => {
    for (const t of timers) clearTimeout(t);
    timers.length = 0;
  };

  const play = (index: number) => {
    if (stopped || index <= highest) return;
    highest = index;
    handlers.onCue(index);
  };

  const runLocally = () => {
    if (stopped || usingFallback) return;
    usingFallback = true;
    handlers.onFallback?.();
    source?.close();
    source = null;
    // only what has not been heard yet, on its own offsets
    const from = highest + 1;
    const base = from > 0 ? cues[highest].at_ms : 0;
    cues.slice(from).forEach((cue, i) => {
      timers.push(setTimeout(() => play(from + i), Math.max(0, cue.at_ms - base)));
    });
    const last = cues.length ? cues[cues.length - 1].at_ms - base : 0;
    timers.push(setTimeout(() => handlers.onAllSent(), Math.max(0, last) + 400));
  };

  const grace = setTimeout(runLocally, STREAM_GRACE_MS);
  timers.push(grace);

  try {
    source = open(`/events?session=${encodeURIComponent(sessionId)}`);
  } catch {
    runLocally();
    return () => {
      stopped = true;
      clearTimers();
    };
  }

  source.addEventListener("hello", () => {
    // the stream is alive, so the local plan is not needed
    clearTimeout(grace);
  });
  source.addEventListener("radio", (e) => {
    if (usingFallback) return;
    clearTimeout(grace);
    try {
      const data = JSON.parse((e as MessageEvent).data) as { index: number };
      play(data.index);
    } catch {
      // a frame we cannot read is one missed line, not a dead night
    }
  });
  source.addEventListener("radio_done", () => {
    if (usingFallback) return;
    handlers.onAllSent();
    // the server has said everything it will ever say on this stream and
    // ends it. Close our side first, so EventSource does not read that end
    // as a failure and reconnect into a finished broadcast
    source?.close();
    source = null;
  });
  source.addEventListener("error", () => {
    // EventSource retries by itself, so an error only matters while
    // nothing has played yet. Once the night is under way, let it
    // reconnect: replayed cues are dropped by play().
    if (highest < 0) runLocally();
  });

  return () => {
    stopped = true;
    clearTimers();
    source?.close();
  };
}

// The slice of EventSource this uses, so a test can hand in its own.
export type EventSourceLike = {
  addEventListener: (type: string, fn: (e: Event) => void) => void;
  close: () => void;
};
