import type { View, Vocalization } from "../engine/run";
import { VOCALIZATION_LABELS } from "../engine/run";
import { forDisplay } from "../engine/display";

// One small component per beat. Presentation only: the engine decides
// what is on the view and the server decides what a signal means.

export function Wake({ view, onNext }: { view: View; onNext: () => void }) {
  return (
    <section className="beat">
      <p className="beat-line">Morning. The floor is cold. Somewhere a bowl clinks.</p>
      <p className="beat-line">
        You are {view.name}. {view.age_group}, {view.breed}.
      </p>
      <p className="beat-line">That is all you know so far.</p>
      <button onClick={onNext}>get up</button>
    </section>
  );
}

export function Scent({ view, onNext }: { view: View; onNext: () => void }) {
  return (
    <section className="beat">
      <p className="beat-line">A trail on the floor. Someone walked here before you.</p>
      <p className="beat-line">Old scent, then fresh, then a bowl. Something with chicken in it.</p>
      {view.scent && <p className="beat-line beat-self">{forDisplay(view.scent.movement)}</p>}
      <button onClick={onNext}>follow it</button>
    </section>
  );
}

export function Visitor({
  view,
  onSignal,
  onNext,
}: {
  view: View;
  onSignal: (v: Vocalization) => void;
  onNext: () => void;
}) {
  const visitor = view.visitor;
  if (!visitor) return null;
  const answered = Boolean(visitor.signal);
  return (
    <section className="beat">
      <p className="beat-line">A pair of shoes stops in front of your kennel.</p>
      <p className="beat-line">A hand rests on the gate. She is looking at you.</p>
      {!answered && (
        <div className="panel" role="group" aria-label="how do you answer">
          {visitor.options.map((v) => (
            <button key={v} onClick={() => onSignal(v)}>
              {VOCALIZATION_LABELS[v]}
            </button>
          ))}
        </div>
      )}
      {answered && visitor.mismatch && (
        <div className="narrator" aria-live="polite">
          <p className="narrator-meant">you meant: {visitor.mismatch.meant}</p>
          <p className="narrator-heard">she heard: {visitor.mismatch.heard}</p>
          <p className="beat-line">She stays a moment longer. Then the shoes move on.</p>
          <button onClick={onNext}>the day goes on</button>
        </div>
      )}
    </section>
  );
}

export function Night({ view, onNext }: { view: View; onNext: () => void }) {
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
      <button onClick={onNext}>sleep</button>
    </section>
  );
}
