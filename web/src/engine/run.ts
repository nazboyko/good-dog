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

export interface ListingRecord {
  age_text?: string;
  weight_text?: string;
  breed: string;
  sex?: string;
  long_stay_evidence?: string;
  quotes: string[];
  description: string;
  default: boolean;
}

export interface EpilogueView {
  name: string;
  photo_url: string;
  photo_width: number;
  photo_height: number;
  listing_url: string;
  org_name: string;
  org_short: string;
  org_city: string;
  org_state: string;
  age_words?: string;
  long_stay: boolean;
  minutes_played: number;
  listing: ListingRecord;
}

export type Ending = "chosen" | "another_dog" | "nobody_today";

export interface View {
  session_id: string;
  day: number;
  beat: Beat;
  ending?: Ending;
  name: string;
  age_group: string;
  breed: string;
  scent?: { movement: string };
  visitor?: {
    arrival: string[];
    options: Vocalization[];
    heard_label: string;
    signal?: Vocalization;
    mismatch?: Mismatch;
    body?: string;
    parting?: string;
  };
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

// fetchRun reads a session by id. Null on any miss: unknown, expired,
// or unreadable. The caller starts a clean run, never an error screen.
export async function fetchRun(sessionId: string): Promise<View | null> {
  const res = await fetch(`/api/session/${sessionId}`);
  if (res.status === 404) return null;
  if (!res.ok) throw new Error((await res.text()).trim() || "something went quiet, try again");
  return (await res.json()) as View;
}

// The session id lives in sessionStorage: a refresh resumes the same
// life, a new tab starts a new one, and it never outlives the browser.
const SESSION_KEY = "good-dog.session";

export function rememberRun(sessionId: string): void {
  try {
    sessionStorage.setItem(SESSION_KEY, sessionId);
  } catch {
    // private mode or storage disabled: the run still plays, it just will not resume
  }
}

export function rememberedRun(): string | null {
  try {
    return sessionStorage.getItem(SESSION_KEY);
  } catch {
    return null;
  }
}

export function forgetRun(): void {
  try {
    sessionStorage.removeItem(SESSION_KEY);
  } catch {
    // nothing to forget
  }
}

export function advance(sessionId: string): Promise<View> {
  return call("POST", `/api/session/${sessionId}/advance`);
}

export function vocalize(sessionId: string, signal: Vocalization): Promise<View> {
  return call("POST", `/api/session/${sessionId}/vocalize`, { signal });
}

// The reveal is a sequence of pauses. Each step is one thing appearing,
// one short line, silence, the next. The photo comes third, after the
// player has been told twice that the dog is real, and the rhythm after
// it matches the rhythm before it. The button is last. Milliseconds,
// tuned by reading the lines out loud.
export const REVEAL_STEPS = [
  { key: "you_were", delay: 0 },
  { key: "not_a_character", delay: 2600 },
  { key: "photo", delay: 2400 },
  { key: "waiting_at", delay: 2600 },
  { key: "age", delay: 2200 },
  { key: "long_stay", delay: 2200 },
  { key: "you_spent", delay: 2200 },
  { key: "meet", delay: 2000 },
] as const;

export type RevealStep = (typeof REVEAL_STEPS)[number]["key"];

// revealSteps returns the steps that will actually render for this dog.
// A dog with no age or no long stay fact skips those lines entirely,
// so no pause is spent on a line that never appears.
export function revealSteps(view: Pick<EpilogueView, "age_words" | "long_stay">) {
  return REVEAL_STEPS.filter((s) => {
    if (s.key === "age") return Boolean(view.age_words);
    if (s.key === "long_stay") return view.long_stay;
    return true;
  });
}

type Steps = readonly { key: RevealStep; delay: number }[];

// revealSchedule returns the absolute time each step appears, so the
// renderer can drive it from one clock and skip ahead on a click.
export function revealSchedule(steps: Steps = REVEAL_STEPS): { key: RevealStep; at: number }[] {
  let t = 0;
  return steps.map((s) => {
    t += s.delay;
    return { key: s.key, at: t };
  });
}

// revealedCount is how many steps are visible at an elapsed time. Steps
// are always a prefix, so order can never be broken.
export function revealedCount(elapsedMs: number, steps: Steps = REVEAL_STEPS): number {
  return revealSchedule(steps).filter((s) => elapsedMs >= s.at).length;
}

// A screen transition breathes: the current screen fades out, a beat of
// empty dark, then the next fades in. Never an instant swap. The fetch
// for the next view runs under the fade out, so the network hides
// inside the motion. Milliseconds.
export const TRANSITION = { out: 360, dark: 240, in: 520 } as const;

// transitionTimes returns how long to hold each phase given whether the
// player prefers reduced motion. Reduced motion swaps instantly, but the
// pressed button still disables so a double click cannot skip a beat.
export function transitionTimes(reducedMotion: boolean): { out: number; dark: number } {
  return reducedMotion ? { out: 0, dark: 0 } : { out: TRANSITION.out, dark: TRANSITION.dark };
}

// easeInOut is the curve the fold follow rides: it starts and ends at
// rest, so the view never lurches. t in [0, 1] to progress in [0, 1].
export function easeInOut(t: number): number {
  const c = Math.min(1, Math.max(0, t));
  return c < 0.5 ? 2 * c * c : 1 - Math.pow(-2 * c + 2, 2) / 2;
}

// followTarget is how far the view must move so a line's bottom edge
// sits just above the fold. Zero when it is already visible, so nothing
// scrolls unless it has to.
export function followTarget(lineBottom: number, viewportHeight: number, margin = 12): number {
  const overflow = lineBottom - (viewportHeight - margin);
  return overflow > 0 ? overflow : 0;
}

// skipAhead returns the elapsed time that shows exactly one more step
// than now, so a click brings the next line forward without shifting
// every later pause. Past the end it returns the time unchanged.
export function skipAhead(elapsedMs: number, steps: Steps = REVEAL_STEPS): number {
  const sched = revealSchedule(steps);
  const shown = revealedCount(elapsedMs, steps);
  if (shown >= sched.length) return elapsedMs;
  return sched[shown].at;
}
