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
// only ever moves earlier. Nothing on this screen is a list.
export function Reveal({ view }: { view: EpilogueView }) {
  const [shown, setShown] = useState(0);
  const [about, setAbout] = useState(false);
  const origin = useRef(performance.now());
  // only the lines this dog will actually get, so no pause is spent on
  // a line that never appears
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

  const has = (key: RevealStep) => {
    const i = steps.findIndex((s) => s.key === key);
    return i >= 0 && shown > i;
  };
  const done = shown >= steps.length;

  if (about) {
    return <AboutListing view={view} onBack={() => setAbout(false)} />;
  }

  return (
    <section className="reveal" onClick={done ? undefined : skip}>
      {has("you_were") && <p className="reveal-line">You were {view.name}.</p>}
      {has("not_a_character") && <p className="reveal-line">{view.name} is not a character.</p>}
      {has("photo") && (
        <figure className="reveal-photo">
          <img src={view.photo_url} alt={`${view.name}, a real dog`} />
        </figure>
      )}
      {has("waiting_at") && (
        <p className="reveal-line">
          {view.name} is real, and waiting with {view.org_short} in {view.org_city}, {view.org_state}.
        </p>
      )}
      {has("age") && (
        <p className="reveal-line">
          {view.name} is {view.age_words}.
        </p>
      )}
      {has("long_stay") && (
        <p className="reveal-line">
          {view.name} is listed among {view.org_short}'s long stay dogs.
        </p>
      )}
      {has("you_spent") && (
        <p className="reveal-line">
          You spent {minutesPhrase(view.minutes_played)} as {view.name}.
        </p>
      )}
      {has("meet") && (
        <div className="reveal-end">
          <a className="reveal-meet" href={view.listing_url} target="_blank" rel="noopener noreferrer">
            meet {view.name}
          </a>
          <button
            type="button"
            className="reveal-about"
            onClick={(e) => {
              e.stopPropagation();
              setAbout(true);
            }}
          >
            About {view.name}, from the listing
          </button>
        </div>
      )}
    </section>
  );
}
