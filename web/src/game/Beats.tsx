import { useEffect, useState } from "react";
import type { View, Vocalization } from "../engine/run";
import { VOCALIZATION_LABELS, closeStagger, prefersReducedMotion } from "../engine/run";
import { listen, play } from "../engine/broadcast";
import { forDisplay } from "../engine/display";

// One small component per beat. Presentation only: the engine decides
// what is on the view and the server decides what a signal means.

export function Wake({ view, onNext, busy }: { view: View; onNext: () => void; busy: boolean }) {
  return (
    <section className="beat">
      <p className="beat-line">Morning. The floor is cold. Somewhere a bowl clinks.</p>
      <p className="beat-line">
        You are {view.name}. {view.age_group}, {view.breed}.
      </p>
      <p className="beat-line">That is all you know so far.</p>
      <button onClick={onNext} disabled={busy}>
        get up
      </button>
    </section>
  );
}

export function Scent({ view, onNext, busy }: { view: View; onNext: () => void; busy: boolean }) {
  return (
    <section className="beat">
      <p className="beat-line">A trail on the floor. Someone walked here before you.</p>
      <p className="beat-line">Old scent, then fresh, then a bowl. Something with chicken in it.</p>
      {view.scent && <p className="beat-line beat-self">{forDisplay(view.scent.movement)}</p>}
      <button onClick={onNext} disabled={busy}>
        follow it
      </button>
    </section>
  );
}

export function Visitor({
  view,
  onSignal,
  onNext,
  busy,
}: {
  view: View;
  onSignal: (v: Vocalization) => void;
  onNext: () => void;
  busy: boolean;
}) {
  const visitor = view.visitor;
  if (!visitor) return null;
  const answered = Boolean(visitor.signal);
  // the visit is several exchanges, only the last one ends the scene
  const lastExchange = visitor.exchange >= visitor.exchanges;
  // the close stages its lines, so the way forward waits for the last one
  const closing = Boolean(visitor.arc || visitor.parting);
  const stagger = closeStagger(prefersReducedMotion());
  const wait = closing ? stagger.button : 0;
  const [ready, setReady] = useState(wait === 0);
  useEffect(() => {
    if (wait === 0) {
      setReady(true);
      return;
    }
    setReady(false);
    const t = window.setTimeout(() => setReady(true), wait);
    return () => window.clearTimeout(t);
  }, [wait, visitor.exchange]);
  // the panel and the narrator share one reserved slot: the panel fades
  // out and the narrator fades in over it, nothing above them moves
  return (
    <section className="beat">
      {visitor.arrival.map((line, i) => (
        <p key={i} className="beat-line">
          {line}
        </p>
      ))}
      <div className="visitor-slot">
        <div
          className={`panel fade-quick ${answered ? "" : "is-shown"}`}
          role="group"
          aria-label="how do you answer"
          aria-hidden={answered}
        >
          {visitor.options.map((v) => (
            <button key={v} onClick={() => onSignal(v)} disabled={busy || answered} tabIndex={answered ? -1 : 0}>
              {VOCALIZATION_LABELS[v]}
            </button>
          ))}
        </div>
        <div className={`narrator fade ${answered ? "is-shown" : ""}`} aria-live="polite" aria-hidden={!answered}>
          {visitor.mismatch && (
            <>
              <div className="narrator-lines">
                <p className="narrator-meant">you meant: {visitor.mismatch.meant}</p>
                <p className="narrator-heard">
                  {visitor.heard_label}: {visitor.mismatch.heard}
                </p>
              </div>
              {visitor.body && <p className="beat-line">{visitor.body}</p>}
              {visitor.arc && (
                <p className="beat-line beat-arc" style={{ animationDelay: `${stagger.arc}ms` }}>
                  {visitor.arc}
                </p>
              )}
              {visitor.parting && (
                <p className="beat-line" style={{ animationDelay: `${stagger.parting}ms` }}>
                  {visitor.parting}
                </p>
              )}
              <button
                className={closing ? "beat-late" : undefined}
                style={closing ? { animationDelay: `${wait}ms` } : undefined}
                onClick={onNext}
                disabled={busy || !ready}
                // the panel hides this whole layer between exchanges, and
                // focus must not be left sitting inside an aria-hidden
                // subtree three times a visit
                tabIndex={answered ? 0 : -1}
              >
                {lastExchange ? "the day goes on" : "and then"}
              </button>
            </>
          )}
        </div>
      </div>
    </section>
  );
}

// Night is the only screen the server drives rather than the player.
// The broadcast arrives a line at a time over the stream, and the
// button waits until the night has finished so nobody sleeps through
// the middle of somebody else's story.
export function Night({ view, onNext, busy }: { view: View; onNext: () => void; busy: boolean }) {
  const cues = view.night?.radio ?? [];
  const [heard, setHeard] = useState(0);
  const [over, setOver] = useState(cues.length === 0);

  useEffect(() => {
    if (cues.length === 0) {
      setOver(true);
      return;
    }
    setHeard(0);
    setOver(false);
    return listen(view.session_id, cues, {
      // cues arrive in order and never twice, so the count is the
      // highest index heard plus one
      onCue: (i) => {
        // the line goes up first, then the voice on top of it, so the
        // night reads the same with the sound off
        setHeard((n) => Math.max(n, i + 1));
        play(cues[i]);
      },
      onDone: () => setOver(true),
    });
    // the broadcast belongs to this session's night, nothing else restarts it
  }, [view.session_id, cues.length]);

  return (
    <section className="beat beat-night">
      <p className="beat-line">Lights out. Somewhere down the hall, a radio.</p>
      <blockquote className="radio">
        {cues.slice(0, heard).map((cue, i) => (
          <p key={i} className={`radio-${cue.speaker}`}>
            {forDisplay(cue.line)}
          </p>
        ))}
      </blockquote>
      <button onClick={onNext} disabled={busy || !over}>
        sleep
      </button>
    </section>
  );
}
