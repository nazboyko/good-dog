import { useEffect, useMemo, useRef, useState } from "react";
import type { EpilogueView, RevealStep } from "../engine/run";
import { easeInOut, followTarget, revealSteps, revealedCount, skipAhead } from "../engine/run";
import { AboutListing } from "./AboutListing";

// the fold follow lasts as long as the quiet fade, so they land together
const FOLLOW_MS = 520;

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
  // back from the panel returns to the reveal at the top of the page,
  // where its centered composition lives
  useEffect(() => {
    if (!about) window.scrollTo(0, 0);
  }, [about]);
  // only the lines this dog will actually get, so no pause is spent on
  // a line that never appears and no space is reserved for one
  const steps = useMemo(() => revealSteps(view), [view]);
  const done = shown >= steps.length;

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

  // safety net for small screens: when a line fades in below the fold the
  // view eases down to it, synced with the fade. any manual scroll cancels
  // the follow for the rest of the reveal, the player is in charge.
  const section = useRef<HTMLElement>(null);
  const following = useRef(true);
  useEffect(() => {
    const cancel = () => {
      following.current = false;
    };
    const onScrollKey = (e: KeyboardEvent) => {
      if (["ArrowDown", "ArrowUp", "PageDown", "PageUp", "Home", "End"].includes(e.key)) cancel();
    };
    window.addEventListener("wheel", cancel, { passive: true });
    window.addEventListener("touchmove", cancel, { passive: true });
    window.addEventListener("keydown", onScrollKey);
    return () => {
      window.removeEventListener("wheel", cancel);
      window.removeEventListener("touchmove", cancel);
      window.removeEventListener("keydown", onScrollKey);
    };
  }, []);
  // one ease at a time: a new line while an ease is running retargets it
  // instead of cancelling it, so a fast click through still arrives
  const ease = useRef<{ from: number; to: number; start: number } | null>(null);
  useEffect(() => {
    if (!following.current || shown === 0 || !section.current) return;
    const el = section.current.querySelector<HTMLElement>(`[data-step="${steps[shown - 1].key}"]`);
    if (!el) return;
    const distance = followTarget(el.getBoundingClientRect().bottom, window.innerHeight);
    if (distance === 0) return;
    const to = window.scrollY + distance;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      window.scrollTo(0, to);
      return;
    }
    if (ease.current) {
      // retarget the running ease from wherever it is now
      ease.current = { ...ease.current, from: window.scrollY, to, start: performance.now() };
      return;
    }
    // eased by hand and timed to the fade, so the view arrives as the
    // line finishes appearing, and it never depends on the browser's
    // smooth scroll, which some states silently drop
    ease.current = { from: window.scrollY, to, start: performance.now() };
    const step = () => {
      const e = ease.current;
      if (!e || !following.current) {
        ease.current = null;
        return;
      }
      const t = (performance.now() - e.start) / FOLLOW_MS;
      window.scrollTo(0, e.from + (e.to - e.from) * easeInOut(t));
      if (t < 1) requestAnimationFrame(step);
      else ease.current = null;
    };
    requestAnimationFrame(step);
  }, [shown, steps]);

  const skip = () => {
    const elapsed = performance.now() - origin.current;
    const ahead = skipAhead(elapsed, steps);
    origin.current -= ahead - elapsed;
    setShown(revealedCount(ahead, steps));
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (about || done) return;
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

  const ratio =
    view.photo_width > 0 && view.photo_height > 0
      ? `${view.photo_width} / ${view.photo_height}`
      : "4 / 3";

  // the panel is its own view: it owns the layout from zero, the reveal
  // is unmounted underneath it, no reserved space leaks across views
  if (about) {
    return <AboutListing view={view} onBack={() => setAbout(false)} />;
  }

  return (
    <>
      <section
        ref={section}
        className="reveal fade is-shown"
        onClick={done ? undefined : skip}
      >
        <p data-step="you_were" className={cls("you_were", "reveal-line")}>
          You were {view.name}.
        </p>
        <p data-step="not_a_character" className={cls("not_a_character", "reveal-line")}>
          {view.name} is not a character.
        </p>
        <figure data-step="photo" className={cls("photo", "reveal-photo")} style={{ aspectRatio: ratio }}>
          <img src={view.photo_url} alt={`${view.name}, a real dog`} />
        </figure>
        <p data-step="waiting_at" className={cls("waiting_at", "reveal-line")}>
          {view.name} is real, and waiting with {view.org_short} in {view.org_city}, {view.org_state}.
        </p>
        {present("age") && (
          <p data-step="age" className={cls("age", "reveal-line")}>
            {view.name} is {view.age_words}.
          </p>
        )}
        {present("long_stay") && (
          <p data-step="long_stay" className={cls("long_stay", "reveal-line")}>
            {view.name} is listed among {view.org_short}'s long stay dogs.
          </p>
        )}
        <p data-step="you_spent" className={cls("you_spent", "reveal-line")}>
          You spent {minutesPhrase(view.minutes_played)} as {view.name}.
        </p>
        <div data-step="meet" className={cls("meet", "reveal-end")} aria-hidden={!has("meet")}>
          <a
            className="btn btn-primary"
            href={view.listing_url}
            target="_blank"
            rel="noopener noreferrer"
            tabIndex={has("meet") ? 0 : -1}
          >
            meet {view.name}
          </a>
          <button
            type="button"
            className="btn-link"
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
    </>
  );
}
