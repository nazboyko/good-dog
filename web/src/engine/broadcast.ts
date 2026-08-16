// The night radio, client side. The server paces the broadcast and this
// only plays what arrives, which is the whole reason the schedule lives
// over there: a backgrounded tab throttles timers, and a night driven
// from the browser would drift out of step with itself.
//
// The stream is the radio. This is the plan for when there is no radio.

export type RadioCue = {
  at_ms: number;
  speaker: "ranger" | "story" | "own";
  line: string;
  // the recording of this line, absent when there is none
  audio?: string;
};

// play starts a cue's recording if it has one. It never blocks the night
// and it never reports failure upward, because the subtitle is the night
// and the voice is on top of it: a browser that refuses to autoplay, a
// muted tab, a missing file and a judge watching with the sound off all
// have to land in exactly the same place, which is a night that reads.
export function play(cue: RadioCue, make: (src: string) => HTMLAudioElement = (src) => new Audio(src)): void {
  if (!cue.audio) return;
  try {
    const sound = make(cue.audio);
    // a promise rejection here is the autoplay policy, not an error
    void sound.play?.()?.catch?.(() => {});
  } catch {
    // no Audio in this environment, the line still shows
  }
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

// How long to wait for the stream to say anything before deciding it is
// not going to. The server sends hello the instant it accepts the
// connection, so silence past this is a real failure, not slowness.
export const STREAM_GRACE_MS = 2500;

export type BroadcastHandlers = {
  onCue: (index: number) => void;
  onDone: () => void;
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
    timers.push(setTimeout(() => handlers.onDone(), Math.max(0, last) + 400));
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
    if (!usingFallback) handlers.onDone();
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
