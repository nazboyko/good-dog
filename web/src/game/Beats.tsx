import type { View, Vocalization } from "../engine/run";
import { VOCALIZATION_LABELS } from "../engine/run";
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
  // the panel and the narrator share one reserved slot: the panel fades
  // out and the narrator fades in over it, nothing above them moves
  return (
    <section className="beat">
      <p className="beat-line">A pair of shoes stops in front of your kennel.</p>
      <p className="beat-line">A hand rests on the gate. She is looking at you.</p>
      <div className="visitor-slot">
        <div
          className={`panel fade-quick ${answered ? "" : "is-shown"}`}
          role="group"
          aria-label="how do you answer"
          aria-hidden={answered}
        >
          {visitor.options.map((v) => (
            <button key={v} onClick={() => onSignal(v)} disabled={busy} tabIndex={answered ? -1 : 0}>
              {VOCALIZATION_LABELS[v]}
            </button>
          ))}
        </div>
        <div className={`narrator fade ${answered ? "is-shown" : ""}`} aria-live="polite" aria-hidden={!answered}>
          {visitor.mismatch && (
            <>
              <p className="narrator-meant">you meant: {visitor.mismatch.meant}</p>
              <p className="narrator-heard">she heard: {visitor.mismatch.heard}</p>
              <p className="beat-line">She stays a moment longer. Then the shoes move on.</p>
              <button onClick={onNext} disabled={busy}>
                the day goes on
              </button>
            </>
          )}
        </div>
      </div>
    </section>
  );
}

export function Night({ view, onNext, busy }: { view: View; onNext: () => void; busy: boolean }) {
  return (
    <section className="beat beat-night">
      <p className="beat-line">Lights out. Somewhere down the hall, a radio.</p>
      {view.night && (
        <blockquote className="radio">
          {view.night.story.map((line, i) => (
            <p key={i} style={{ animationDelay: `${i * 1.6}s` }}>
              {forDisplay(line)}
            </p>
          ))}
        </blockquote>
      )}
      <button onClick={onNext} disabled={busy}>
        sleep
      </button>
    </section>
  );
}
