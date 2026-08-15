import { useEffect, useRef, useState } from "react";
import type { EpilogueView } from "../engine/run";
import { REVEAL_STEPS, revealedCount, skipAhead } from "../engine/run";
import { forDisplay } from "../engine/display";

export function minutesPhrase(minutes: number): string {
  if (minutes < 1) return "less than a minute";
  if (minutes === 1) return "one minute";
  return `${minutes} minutes`;
}

// The reveal. Pauses, not exclamation points. Each line appears on its
// own, the photo appears here for the first time in the whole game, and
// a click or key brings the next line forward without shifting the
// pauses after it. One clock, its origin only ever moves earlier.
export function Reveal({ view }: { view: EpilogueView }) {
  const [shown, setShown] = useState(0);
  const origin = useRef(performance.now());

  useEffect(() => {
    // the route is open now, so the photo can load before its moment
    const img = new Image();
    img.src = view.photo_url;
  }, [view.photo_url]);

  useEffect(() => {
    let frame = 0;
    const tick = () => {
      const count = revealedCount(performance.now() - origin.current);
      setShown((prev) => (prev === count ? prev : count));
      if (count < REVEAL_STEPS.length) frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [shown]);

  const skip = () => {
    const elapsed = performance.now() - origin.current;
    const ahead = skipAhead(elapsed);
    origin.current -= ahead - elapsed;
    setShown(revealedCount(ahead));
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === " " || e.key === "Enter" || e.key === "ArrowRight") {
        e.preventDefault();
        skip();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const has = (i: number) => shown > i;

  return (
    <section className="reveal" onClick={skip}>
      {has(0) && <p className="reveal-line">You were {view.name}.</p>}
      {has(1) && <p className="reveal-line">{view.name} is not a character.</p>}
      {has(2) && (
        <figure className="reveal-photo">
          <img src={view.photo_url} alt={`${view.name}, a real dog`} />
        </figure>
      )}
      {has(3) && (
        <div className="reveal-facts">
          <p>
            {view.name} is with {view.org_name} in {view.org_city}, {view.org_state}.
          </p>
          {view.age_text && <p>Age on the listing: {view.age_text}.</p>}
          {view.weight_text && <p>Weight on the listing: {view.weight_text}.</p>}
          {view.quotes.length > 0 && (
            <>
              <p className="reveal-quotes-lead">From the listing:</p>
              <ul className="reveal-quotes">
                {view.quotes.map((q) => (
                  <li key={q}>{forDisplay(q)}</li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
      {has(4) && (
        <p className="reveal-line">
          You spent {minutesPhrase(view.minutes_played)} as {view.name}.
          {view.long_stay && ` ${view.name} is on the shelter's long stay list.`}
          {!view.long_stay && ` ${view.name} is still there.`}
        </p>
      )}
      {has(5) && (
        <a className="reveal-meet" href={view.listing_url} target="_blank" rel="noopener noreferrer">
          meet {view.name}
        </a>
      )}
    </section>
  );
}
