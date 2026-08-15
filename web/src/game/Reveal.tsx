import { useEffect, useMemo, useRef, useState } from "react";
import type { EpilogueView, RevealStep } from "../engine/run";
import { revealSteps, revealedCount, skipAhead } from "../engine/run";
import { AboutListing } from "./AboutListing";

export function minutesPhrase(minutes: number): string {
  if (minutes < 1) return "less than a minute";
  if (minutes === 1) return "one minute";
  return `${minutes} minutes`;
}

// The reveal. Pauses, not exclamation points. Short lines, silence
// between them, the photo appears here for the first time in the whole
// game, and the button comes last. A click or key brings the next line
// forward without shifting the pauses after it. One clock, its origin
// only ever moves earlier.
//
// Every element is in the DOM from the first frame with its space
// reserved. Appearing is a class that changes opacity and nothing else,
// so nothing above a new line moves by a pixel. Reserved space plus
// fade, never insert and push.
export function Reveal({ view }: { view: EpilogueView }) {
  const [shown, setShown] = useState(0);
  const [about, setAbout] = useState(false);
  const origin = useRef(performance.now());
  // only the lines this dog will actually get, so no pause is spent on
  // a line that never appears and no space is reserved for one
  const steps = useMemo(() => revealSteps(view), [view]);

  useEffect(() => {
    // the route is open now, so the photo can load before its moment
    const img = new Image();
    img.src = view.photo_url;
  }, [view.photo_url]);

  useEffect(() => {
    let frame = 0;
    const tick = () => {
      const count = revealedCount(performance.now() - origin.current, steps);
      setShown((prev) => (prev === count ? prev : count));
      if (count < steps.length) frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [shown, steps]);

  const skip = () => {
    const elapsed = performance.now() - origin.current;
    const ahead = skipAhead(elapsed, steps);
    origin.current -= ahead - elapsed;
    setShown(revealedCount(ahead, steps));
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (about) return;
      if (e.key === " " || e.key === "Enter" || e.key === "ArrowRight") {
        e.preventDefault();
        skip();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  // present means the dog gets this line, has means it is visible now
  const present = (key: RevealStep) => steps.some((s) => s.key === key);
  const has = (key: RevealStep) => {
    const i = steps.findIndex((s) => s.key === key);
    return i >= 0 && shown > i;
  };
  const cls = (key: RevealStep, base: string) => `${base} fade ${has(key) ? "is-shown" : ""}`;
  const done = shown >= steps.length;

  const ratio =
    view.photo_width > 0 && view.photo_height > 0
      ? `${view.photo_width} / ${view.photo_height}`
      : "4 / 3";

  return (
    <>
      <section
        className={`reveal fade ${about ? "" : "is-shown"}`}
        onClick={done ? undefined : skip}
        aria-hidden={about}
      >
        <p className={cls("you_were", "reveal-line")}>You were {view.name}.</p>
        <p className={cls("not_a_character", "reveal-line")}>{view.name} is not a character.</p>
        <figure className={cls("photo", "reveal-photo")} style={{ aspectRatio: ratio }}>
          <img src={view.photo_url} alt={`${view.name}, a real dog`} />
        </figure>
        <p className={cls("waiting_at", "reveal-line")}>
          {view.name} is real, and waiting with {view.org_short} in {view.org_city}, {view.org_state}.
        </p>
        {present("age") && (
          <p className={cls("age", "reveal-line")}>
            {view.name} is {view.age_words}.
          </p>
        )}
        {present("long_stay") && (
          <p className={cls("long_stay", "reveal-line")}>
            {view.name} is listed among {view.org_short}'s long stay dogs.
          </p>
        )}
        <p className={cls("you_spent", "reveal-line")}>
          You spent {minutesPhrase(view.minutes_played)} as {view.name}.
        </p>
        <div className={cls("meet", "reveal-end")} aria-hidden={!has("meet")}>
          <a
            className="reveal-meet"
            href={view.listing_url}
            target="_blank"
            rel="noopener noreferrer"
            tabIndex={has("meet") ? 0 : -1}
          >
            meet {view.name}
          </a>
          <button
            type="button"
            className="reveal-about"
            tabIndex={has("meet") ? 0 : -1}
            onClick={(e) => {
              e.stopPropagation();
              setAbout(true);
            }}
          >
            About {view.name}, from the listing
          </button>
        </div>
      </section>
      <div className={`about-wrap fade ${about ? "is-shown" : ""}`} aria-hidden={!about}>
        {about && <AboutListing view={view} onBack={() => setAbout(false)} />}
      </div>
    </>
  );
}
