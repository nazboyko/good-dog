import { expect, test } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { readdirSync } from "node:fs";
import { SceneBackdrop } from "./Scene";
import { REVEAL_BEATS, sceneFor } from "../engine/scenes";
import type { Beat } from "../engine/run";

// The hard boundary. Dog vision is the whole run and it ends at the
// epilogue: the real photo is the first true image in the game, and it
// only reads that way if nothing before the reveal was true color and
// nothing at the reveal is anything but.
//
// These render the component the way the shell does. renderToStaticMarkup
// runs the render pass without effects, which is exactly the question
// being asked: not "did the shader run" but "is there a canvas on this
// screen at all". If no canvas is emitted, nothing can be shaded.

test("no shaded canvas is ever mounted under the reveal", () => {
  for (const beat of REVEAL_BEATS) {
    const markup = renderToStaticMarkup(<SceneBackdrop beat={beat} />);
    expect(markup, `beat ${beat} put something behind the reveal`).toBe("");
  }
});

test("the beats inside the dog's day do get a scene", () => {
  for (const beat of ["wake", "scent", "visitor"] as Beat[]) {
    const markup = renderToStaticMarkup(<SceneBackdrop beat={beat} />);
    expect(markup, `beat ${beat} lost its room`).toContain("<canvas");
  }
});

// A beat nobody thought about must not quietly inherit a scene. The
// enforcement is the Record<Beat, ...> in scenes.ts, which is a compile
// error when the vocabulary grows. This only pins the boundary itself,
// which is a value and so needs a runtime check.
test("the reveal beats never resolve to a scene", () => {
  for (const beat of REVEAL_BEATS) {
    expect(sceneFor(beat), `${beat} must never resolve to a scene`).toBeNull();
  }
});

// The reveal renders the real photo as a plain img, never into a canvas,
// so no pass can sit between the shelter's picture and the player.
test("the reveal shows the photo as an untouched image", async () => {
  const { Reveal } = await import("./Reveal");
  const markup = renderToStaticMarkup(
    <Reveal
      view={{
        name: "Venus",
        photo_url: "/photos/venus.jpg",
        photo_width: 800,
        photo_height: 600,
        listing_url: "https://example.org/venus",
        org_name: "Ruff Start Rescue",
        org_short: "Ruff Start Rescue",
        org_city: "Princeton",
        org_state: "Minnesota",
        age_words: "four years old",
        reality_line: "Venus is real, and waiting with Ruff Start Rescue in Princeton, Minnesota.",
        seam: false,
        adopted: false,
        long_stay: true,
        minutes_played: 12,
        listing: {
          breed: "Pit Bull Terrier mix",
          quotes: [],
          description: "",
          default: false,
          retrieved_on: "August 15, 2026",
        },
      }}
    />,
  );
  expect(markup).not.toContain("<canvas");
});

// Every beat the dog lives through has its own room, and the night and
// the meeting room are not the daytime row. Ground truth is the file
// list under web/public/scenes, not this table repeated.
test("each beat maps to its own room and every room exists", async () => {
  const { sceneFor, veilFor } = await import("../engine/scenes");
  const onDisk = new Set(readdirSync("public/scenes").map((f) => `/scenes/${f}`));

  const rooms: Record<string, string | null> = {
    wake: sceneFor("wake"),
    scent: sceneFor("scent"),
    visitor: sceneFor("visitor"),
    night: sceneFor("night"),
    adoption: sceneFor("adoption"),
  };
  for (const [beat, src] of Object.entries(rooms)) {
    expect(src, `${beat} has no room`).toBeTruthy();
    expect(onDisk.has(src!), `${beat} points at ${src}, which is not on disk`).toBe(true);
  }
  // Which room, by name. Distinct is not enough: the yard and the
  // corridor are on disk too and would pass a distinctness check, and
  // the meeting room is the point of day three because it has no gate.
  expect(rooms.night).toBe("/scenes/night-enclosure.webp");
  expect(rooms.adoption).toBe("/scenes/dating-room.webp");
  // and the day beats are all the same row
  expect(rooms.wake).toBe("/scenes/enclosure.webp");
  expect(rooms.scent).toBe("/scenes/enclosure.webp");
  expect(rooms.visitor).toBe("/scenes/enclosure.webp");

  // Each room carries a veil measured against itself, and the floor is
  // per room, written down here rather than read from the table under
  // test. The numbers are the ones measured in the browser against the
  // brightest part of each room's middle band: muted text has to land
  // at least where it lands on the enclosure, 3.34 to 1.
  //   enclosure       0.80 -> 3.34, the baseline the copy was tuned on
  //   night enclosure 0.80 -> 3.78, at 0.55 it was 1.83
  //   meeting room    0.86 -> 4.06, at 0.80 it was 3.29, under the floor
  // A room falling back to a default is a room nobody measured, so the
  // meeting room's floor sits above the fallback on purpose.
  const floors: Record<string, number> = {
    "/scenes/enclosure.webp": 0.8,
    "/scenes/night-enclosure.webp": 0.8,
    "/scenes/dating-room.webp": 0.86,
  };
  for (const [src, floor] of Object.entries(floors)) {
    expect(veilFor(src), `${src} veil below its measured floor`).toBeGreaterThanOrEqual(floor);
    expect(veilFor(src)).toBeLessThanOrEqual(0.95);
  }
});
