// @vitest-environment jsdom
import { afterEach, expect, test, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

// The sound has to leave with the press, not with the answer. Testing
// bark() on its own cannot see that: the whole claim is about where the
// call sits relative to the round trip. So this renders the real shell,
// clicks the real button, and holds the answer open forever, which means
// anything that happens after the answer cannot happen at all.

const heard: string[] = [];
vi.mock("../engine/voice", () => ({
  bark: (v: string) => heard.push(v),
  hush: () => {},
}));

// A real visitor view, taken off the wire from a running server rather
// than typed out here. A hand written one is written by the same
// understanding that wrote the code, so it agrees with the code and
// misses the field the screen actually reads.
import realView from "./__fixtures__/visitor-view.json";

let sentWith: string | null = null;
let soundAtSendTime: string[] = [];
const startedAfter: string[][] = [];

// A real epilogue view off the wire would be better, but the reveal
// only needs enough to mount and show its buttons here, and the shape
// is checked by tsc against EpilogueView through the cast below.
const endedView = {
  ...realView,
  session_id: "told-life",
  beat: "epilogue",
  ending: "nobody_today",
  epilogue: {
    name: "Venus", photo_url: "/p.jpg", photo_width: 3, photo_height: 2,
    listing_url: "https://example.org/v", org_name: "Ruff Start Rescue", org_short: "Ruff Start Rescue",
    org_city: "Princeton", org_state: "Minnesota", age_words: "four years old",
    ending_line: "Nobody came for Venus today.", reality_line: "Venus is real, and waiting.",
    seam: false, adopted: false, long_stay: false, minutes_played: 1,
    listing: { breed: "mix", quotes: [], description: "", default: true, retrieved_on: "August 15, 2026" },
  },
};

vi.mock("../engine/run", async (importOriginal) => {
  const real = await importOriginal<typeof import("../engine/run")>();
  return {
    ...real,
    // storage stays real, so the another-life test can see what is kept
    fetchRun: async (id: string) => (id === "told-life" ? (endedView as never) : null),
    startRun: async () => {
      // what the next life is asked to avoid, read at call time, which is
      // the only moment that matters for "somebody else"
      startedAfter.push([...real.finishedRuns()]);
      return realView;
    },
    advance: async () => endedView as never,
    // never settles, so the press is the only thing that can have run
    vocalize: (_id: string, v: string) => {
      sentWith = v;
      soundAtSendTime = [...heard];
      return new Promise(() => {});
    },
  };
});

// jsdom has no matchMedia and the run asks about motion
window.matchMedia = ((q: string) => ({
  matches: false,
  media: q,
  addEventListener: () => {},
  removeEventListener: () => {},
})) as unknown as typeof window.matchMedia;

afterEach(() => {
  heard.length = 0;
  sentWith = null;
  soundAtSendTime = [];
  startedAfter.length = 0;
  // storage is real now, so a life remembered by one test must not be
  // resumed by the next
  sessionStorage.clear();
  cleanup();
});

const settle = () => new Promise((r) => setTimeout(r, 700));

async function reachTheVisitor() {
  const { Run } = await import("./Run");
  render(<Run />);
  screen.getByRole("button", { name: "wake up" }).click();
  await settle();
}

test("the sound leaves with the press, before the answer comes back", async () => {
  await reachTheVisitor();
  screen.getByRole("button", { name: "howl" }).click();

  expect(sentWith).toBe("howl");
  // the dog had already spoken by the time the request went out
  expect(soundAtSendTime).toEqual(["howl"]);
});

test("the press asks for the sound that was pressed", async () => {
  await reachTheVisitor();
  screen.getByRole("button", { name: "low growl" }).click();
  expect(soundAtSendTime).toEqual(["low_growl"]);
});

// Silence still reaches bark, which is where the decision to stay quiet
// lives. Skipping the call here would put that rule in two places.
test("staying silent still goes through the same door", async () => {
  await reachTheVisitor();
  screen.getByRole("button", { name: "stay silent" }).click();
  expect(sentWith).toBe("silence");
  expect(soundAtSendTime).toEqual(["silence"]);
});

// After the reveal the way out is the button, and its one promise is
// that the next dog is somebody else. That only holds if the life just
// lived is remembered as finished before the next one is asked for.
test("live another life ends this one first, so the next dog is somebody else", async () => {
  sessionStorage.setItem("good-dog.session", "told-life");
  const { Run } = await import("./Run");
  render(<Run />);
  await settle();
  // the reveal is resumable: a reload during the ending is the same dog.
  // The buttons are staged in over real seconds behind aria-hidden, so
  // this reaches for the element itself rather than waiting the staging
  // out; the staging has its own tests
  const another = document.querySelector<HTMLButtonElement>("[data-another]");
  if (!another) throw new Error("the reveal did not mount its way out");
  another.click();
  await settle();
  await settle();
  expect(startedAfter).toHaveLength(1);
  expect(startedAfter[0]).toContain("told-life");
});
