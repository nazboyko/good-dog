import { expect, test } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
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
        still_waiting: true,
        long_stay: true,
        minutes_played: 12,
        listing: {
          breed: "Pit Bull Terrier mix",
          quotes: [],
          description: "",
          default: false,
        },
      }}
    />,
  );
  expect(markup).not.toContain("<canvas");
});
