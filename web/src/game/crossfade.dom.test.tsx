// @vitest-environment jsdom
import { afterEach, expect, test, vi } from "vitest";
import { act, cleanup, render } from "@testing-library/react";
import { SceneBackdrop } from "./Scene";
import { CROSSFADE_MS } from "../engine/scenes";

// A paint the test controls: every room's paint stays pending until the
// test resolves it, which is the only way to see what is on screen while
// a room is still being drawn.
function controllablePaint() {
  const pending = new Map<string, () => void>();
  const paint = (_c: HTMLCanvasElement, src: string) =>
    new Promise<void>((resolve) => {
      pending.set(src, resolve);
    });
  return { paint, finish: (src: string) => act(async () => { pending.get(src)?.(); }) };
}

const rooms = (c: HTMLElement) => [...c.querySelectorAll(".scene-room")].map((r) => r.getAttribute("data-room")!.split("/").pop());
const shown = (c: HTMLElement) => [...c.querySelectorAll(".scene-room.is-shown")].map((r) => r.getAttribute("data-room")!.split("/").pop());

afterEach(cleanup);

// The whole point: the old room stays until the new one has painted.
// Dropping it on the beat change shows bare ground for as long as the
// paint takes, which on a phone is long enough to see.
test("the old room holds until the new one has painted, then lets go", async () => {
  vi.useFakeTimers();
  const { paint, finish } = controllablePaint();
  const { container, rerender } = render(<SceneBackdrop beat="visitor" paint={paint} />);
  await finish("/scenes/enclosure.webp");
  expect(shown(container)).toEqual(["enclosure.webp"]);

  // the beat changes and the night starts painting
  rerender(<SceneBackdrop beat="night" paint={paint} />);
  expect(rooms(container)).toEqual(["enclosure.webp", "night-enclosure.webp"]);
  // the day room is still shown, the night is not yet
  expect(shown(container)).toEqual(["enclosure.webp"]);

  // the night paints: both on screen for the length of the fade
  await finish("/scenes/night-enclosure.webp");
  expect(shown(container)).toEqual(["enclosure.webp", "night-enclosure.webp"]);
  await act(async () => { vi.advanceTimersByTime(CROSSFADE_MS - 10); });
  expect(rooms(container)).toHaveLength(2);

  // and then the day room is gone
  await act(async () => { vi.advanceTimersByTime(20); });
  expect(rooms(container)).toEqual(["night-enclosure.webp"]);
  vi.useRealTimers();
});

// Going back to a room that is still mounted underneath: it never
// repaints, so a release driven only by the paint callback strands the
// other layer under it for ever.
test("returning to a mounted room still releases the layer underneath", async () => {
  vi.useFakeTimers();
  const { paint, finish } = controllablePaint();
  const { container, rerender } = render(<SceneBackdrop beat="visitor" paint={paint} />);
  await finish("/scenes/enclosure.webp");
  rerender(<SceneBackdrop beat="night" paint={paint} />);
  await finish("/scenes/night-enclosure.webp");
  // before the fade has released the day room, the beat goes back to it
  rerender(<SceneBackdrop beat="wake" paint={paint} />);
  expect(rooms(container)).toEqual(["night-enclosure.webp", "enclosure.webp"]);
  await act(async () => { vi.advanceTimersByTime(CROSSFADE_MS + 10); });
  expect(rooms(container)).toEqual(["enclosure.webp"]);
  vi.useRealTimers();
});

// A room that fails to paint must not hold the old one on screen for
// ever: the fallback is dark ground, not a frozen crossfade.
test("a room that cannot paint still lets the change through", async () => {
  vi.useFakeTimers();
  const failing = (_c: HTMLCanvasElement, src: string) =>
    src.includes("night") ? Promise.reject(new Error("no webgl")) : Promise.resolve();
  const errors = vi.spyOn(console, "error").mockImplementation(() => {});
  const { container, rerender } = render(<SceneBackdrop beat="visitor" paint={failing} />);
  await act(async () => {});
  rerender(<SceneBackdrop beat="night" paint={failing} />);
  await act(async () => {});
  await act(async () => { vi.advanceTimersByTime(CROSSFADE_MS + 10); });
  expect(rooms(container)).toEqual(["night-enclosure.webp"]);
  errors.mockRestore();
  vi.useRealTimers();
});

// The boundary, restated for the crossfade: leaving for the reveal
// unmounts every room, it does not crossfade to nothing.
test("the reveal has no room at all, not a fading one", async () => {
  const { paint, finish } = controllablePaint();
  const { container, rerender } = render(<SceneBackdrop beat="adoption" paint={paint} />);
  await finish("/scenes/dating-room.webp");
  rerender(<SceneBackdrop beat="epilogue" paint={paint} />);
  await act(async () => {});
  expect(container.querySelectorAll("canvas")).toHaveLength(0);
});
