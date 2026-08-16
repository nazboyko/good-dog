import { expect, test } from "vitest";
import {
  REVEAL_STEPS,
  revealSchedule,
  revealSteps,
  revealedCount,
  skipAhead,
  VOCALIZATION_LABELS,
} from "./run";

test("the reveal shows the photo only after the player is told twice the dog is real", () => {
  const keys = REVEAL_STEPS.map((s) => s.key);
  const photoAt = keys.indexOf("photo");
  expect(keys.indexOf("you_were")).toBeLessThan(photoAt);
  expect(keys.indexOf("not_a_character")).toBeLessThan(photoAt);
  expect(keys[keys.length - 1]).toBe("meet");
});

test("after the photo the lines keep the opening rhythm, one at a time, the button last", () => {
  // the common path: she is still waiting, so there is no seam to name
  const steps = revealSteps({ age_words: "four years old", long_stay: true, still_waiting: true });
  const keys = steps.map((s) => s.key);
  expect(keys.slice(keys.indexOf("photo") + 1)).toEqual([
    "waiting_at",
    "age",
    "long_stay",
    "you_spent",
    "meet",
  ]);
  const opening = REVEAL_STEPS[1].delay;
  for (const step of REVEAL_STEPS.slice(3)) {
    // no line after the photo lands faster than the opening beat minus a breath
    expect(step.delay).toBeGreaterThanOrEqual(opening - 700);
  }
});

test("reveal schedule is cumulative and never rushes", () => {
  const sched = revealSchedule();
  expect(sched[0].at).toBe(0);
  for (let i = 1; i < sched.length; i++) {
    expect(sched[i].at).toBeGreaterThan(sched[i - 1].at);
    // no step lands less than a second and a half after the last
    expect(sched[i].at - sched[i - 1].at).toBeGreaterThanOrEqual(1500);
  }
});

test("a click at 900ms shows the second line now and the photo 2.4s later, never everything at once", () => {
  // simulate the component: one clock origin, a click moves the origin
  // back by exactly the gap to the next step, then time keeps flowing
  let origin = 0;
  const now = (t: number) => t - origin;
  expect(revealedCount(now(900))).toBe(1);
  const elapsed = now(900);
  const ahead = skipAhead(elapsed);
  origin -= ahead - elapsed;
  expect(revealedCount(now(900))).toBe(2);
  // 100ms after the click nothing else has appeared
  expect(revealedCount(now(1000))).toBe(2);
  // the photo lands its own delay after the click, at 900 + 2400
  expect(revealedCount(now(900 + 2400 - 1))).toBe(2);
  expect(revealedCount(now(900 + 2400))).toBe(3);
  // and meet lands at the sum of the remaining delays, not later
  const remaining = REVEAL_STEPS.slice(2).reduce((s, x) => s + x.delay, 0);
  expect(revealedCount(now(900 + remaining - 1))).toBe(REVEAL_STEPS.length - 1);
  expect(revealedCount(now(900 + remaining))).toBe(REVEAL_STEPS.length);
});

test("skipping ahead brings one line forward and keeps every later pause", () => {
  const sched = revealSchedule();
  // at 1s only the first line is up, a click shows the second right now
  expect(revealedCount(1000)).toBe(1);
  const afterClick = skipAhead(1000);
  expect(revealedCount(afterClick)).toBe(2);
  // and the photo still arrives exactly its own delay after that, not later
  expect(sched[2].at - afterClick).toBe(REVEAL_STEPS[2].delay);
  // clicking through everything shows every step, then stays put
  let t = 0;
  for (let i = 0; i < 12; i++) t = skipAhead(t);
  expect(revealedCount(t)).toBe(sched.length);
  expect(skipAhead(t)).toBe(t);
});

test("a dog with no long stay fact and no age spends no pause on those lines", () => {
  const steps = revealSteps({ age_words: "", long_stay: false, still_waiting: true });
  const keys = steps.map((s) => s.key);
  expect(keys).not.toContain("age");
  expect(keys).not.toContain("long_stay");
  expect(keys[keys.length - 1]).toBe("meet");
  // the you_spent line follows the waiting line by exactly its own delay
  const sched = revealSchedule(steps);
  const waitingAt = sched.find((s) => s.key === "waiting_at")!.at;
  const spentAt = sched.find((s) => s.key === "you_spent")!.at;
  expect(spentAt - waitingAt).toBe(REVEAL_STEPS.find((s) => s.key === "you_spent")!.delay);
  // and every step counts toward the click through, none is a phantom
  let t = 0;
  for (let i = 0; i < steps.length; i++) t = skipAhead(t, steps);
  expect(revealedCount(t, steps)).toBe(steps.length);
});

test("every vocalization has a plain label and silence is a real choice", () => {
  const labels = Object.values(VOCALIZATION_LABELS);
  expect(labels).toHaveLength(6);
  for (const label of labels) {
    expect(label).toBe(label.toLowerCase());
    expect(label).not.toMatch(/[—–]/);
  }
  expect(VOCALIZATION_LABELS.silence).toContain("silent");
});

test("minutes phrase reads like a person wrote it", async () => {
  const { minutesPhrase } = await import("../game/Reveal");
  expect(minutesPhrase(0)).toBe("less than a minute");
  expect(minutesPhrase(1)).toBe("one minute");
  expect(minutesPhrase(38)).toBe("38 minutes");
});

test("the fold follow moves only as far as needed and rides an eased curve", async () => {
  const { easeInOut, followTarget } = await import("./run");
  // a line already on screen asks for no movement at all
  expect(followTarget(500, 800)).toBe(0);
  expect(followTarget(788, 800)).toBe(0);
  // a line 61px below the fold asks for 61 plus the margin
  expect(followTarget(861, 800)).toBe(73);
  // the curve rests at both ends and is monotonic in between
  expect(easeInOut(0)).toBe(0);
  expect(easeInOut(1)).toBe(1);
  expect(easeInOut(0.5)).toBe(0.5);
  let last = 0;
  for (let t = 0; t <= 1; t += 0.05) {
    const v = easeInOut(t);
    expect(v).toBeGreaterThanOrEqual(last);
    last = v;
  }
  // and it clamps outside the range, so an overshoot frame never lurches
  expect(easeInOut(1.4)).toBe(1);
  expect(easeInOut(-0.2)).toBe(0);
});

test("screen transitions breathe: out, a beat of dark, in, and reduced motion swaps instantly", async () => {
  const { TRANSITION, transitionTimes } = await import("./run");
  // the ranges the design asks for
  expect(TRANSITION.out).toBeGreaterThanOrEqual(300);
  expect(TRANSITION.out).toBeLessThanOrEqual(400);
  expect(TRANSITION.dark).toBeGreaterThanOrEqual(200);
  expect(TRANSITION.dark).toBeLessThanOrEqual(300);
  // leaving is quicker than arriving
  expect(TRANSITION.out).toBeLessThan(TRANSITION.in);
  // reduced motion: no waits at all, the swap is instant
  expect(transitionTimes(true)).toEqual({ out: 0, dark: 0 });
  expect(transitionTimes(false)).toEqual({ out: TRANSITION.out, dark: TRANSITION.dark });
});

// The seam. After a chosen ending the game has just said she went home
// and the reveal is about to say she is still listed. One line takes
// the turn on itself, so the player is told the frame rather than left
// to discover a contradiction, which is what would read as a trick.
test("the seam line appears only on the path where the ending and the listing disagree", () => {
  const chosen = revealSteps({ age_words: "four years old", long_stay: true, still_waiting: false });
  const waiting = revealSteps({ age_words: "four years old", long_stay: true, still_waiting: true });

  expect(chosen.map((s) => s.key)).toContain("your_three_days");
  expect(waiting.map((s) => s.key)).not.toContain("your_three_days");

  // it lands after the photo and before the line it is framing
  const keys = chosen.map((s) => s.key);
  expect(keys.indexOf("your_three_days")).toBeGreaterThan(keys.indexOf("photo"));
  expect(keys.indexOf("your_three_days")).toBeLessThan(keys.indexOf("waiting_at"));

  // and it costs a real pause, it is not a phantom step
  const sched = revealSchedule(chosen);
  const photoAt = sched.find((s) => s.key === "photo")!.at;
  const seamAt = sched.find((s) => s.key === "your_three_days")!.at;
  expect(seamAt).toBeGreaterThan(photoAt);
  // every step still counts toward the click through
  let t = 0;
  for (let i = 0; i < chosen.length; i++) t = skipAhead(t, chosen);
  expect(revealedCount(t, chosen)).toBe(chosen.length);
});
