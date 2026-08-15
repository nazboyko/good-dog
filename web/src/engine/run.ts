// The run engine talks to the server session and hands React a View to
// draw. Beat order, what is visible, and what a signal means all live on
// the server. This file only asks and renders. Plain TypeScript, no React.

export type Beat = "wake" | "scent" | "visitor" | "night" | "epilogue" | "done";

export type Vocalization =
  | "playful_bark"
  | "alert_bark"
  | "whine"
  | "low_growl"
  | "howl"
  | "silence";

export interface Mismatch {
  meant: string;
  heard: string;
}

export interface EpilogueView {
  name: string;
  photo_url: string;
  listing_url: string;
  org_name: string;
  org_city: string;
  org_state: string;
  age_text?: string;
  weight_text?: string;
  long_stay: boolean;
  minutes_played: number;
  quotes: string[];
}

export interface View {
  session_id: string;
  beat: Beat;
  name: string;
  age_group: string;
  breed: string;
  scent?: { movement: string };
  visitor?: { options: Vocalization[]; signal?: Vocalization; mismatch?: Mismatch };
  night?: { story: string[] };
  epilogue?: EpilogueView;
}

// The panel labels are player facing copy. Silence is a real choice.
export const VOCALIZATION_LABELS: Record<Vocalization, string> = {
  playful_bark: "playful bark",
  alert_bark: "alert bark",
  whine: "whine",
  low_growl: "low growl",
  howl: "howl",
  silence: "stay silent",
};

async function call(method: string, path: string, body?: unknown): Promise<View> {
  const res = await fetch(path, {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    // the server's words are already warm, the player never sees a number
    throw new Error((await res.text()).trim() || "something went quiet, try again");
  }
  return (await res.json()) as View;
}

export function startRun(): Promise<View> {
  return call("POST", "/api/session");
}

export function advance(sessionId: string): Promise<View> {
  return call("POST", `/api/session/${sessionId}/advance`);
}

export function vocalize(sessionId: string, signal: Vocalization): Promise<View> {
  return call("POST", `/api/session/${sessionId}/vocalize`, { signal });
}

// The reveal is a sequence of pauses. Each step is one thing appearing.
// The photo comes third, after the player has been told twice that the
// dog is real. Numbers are in milliseconds and were tuned by reading the
// lines out loud.
export const REVEAL_STEPS = [
  { key: "you_were", delay: 0 },
  { key: "not_a_character", delay: 2600 },
  { key: "photo", delay: 2400 },
  { key: "facts", delay: 2200 },
  { key: "waiting", delay: 1800 },
  { key: "meet", delay: 1600 },
] as const;

export type RevealStep = (typeof REVEAL_STEPS)[number]["key"];

// revealSchedule returns the absolute time each step appears, so the
// renderer can drive it from one clock and skip ahead on a click.
export function revealSchedule(): { key: RevealStep; at: number }[] {
  let t = 0;
  return REVEAL_STEPS.map((s) => {
    t += s.delay;
    return { key: s.key, at: t };
  });
}

// revealedCount is how many steps are visible at an elapsed time. Steps
// are always a prefix, so order can never be broken.
export function revealedCount(elapsedMs: number): number {
  return revealSchedule().filter((s) => elapsedMs >= s.at).length;
}

// skipAhead returns the elapsed time that shows exactly one more step
// than now, so a click brings the next line forward without shifting
// every later pause. Past the end it returns the time unchanged.
export function skipAhead(elapsedMs: number): number {
  const sched = revealSchedule();
  const shown = revealedCount(elapsedMs);
  if (shown >= sched.length) return elapsedMs;
  return sched[shown].at;
}
