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

vi.mock("../engine/run", async (importOriginal) => {
  const real = await importOriginal<typeof import("../engine/run")>();
  return {
    ...real,
    rememberedRun: () => null,
    rememberRun: () => {},
    forgetRun: () => {},
    startRun: async () => realView,
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
