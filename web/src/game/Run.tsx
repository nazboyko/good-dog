import { useState } from "react";
import type { View, Vocalization } from "../engine/run";
import { advance, startRun, vocalize } from "../engine/run";
import { Night, Scent, Visitor, Wake } from "./Beats";
import { Reveal } from "./Reveal";
import { useViewTop } from "./useViewTop";

// The run shell. Holds the current view and calls the engine. Which
// beat comes next is the server's decision, this only asks.
export function Run() {
  const [view, setView] = useState<View | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const guard = async (work: () => Promise<View>) => {
    if (busy) return;
    setBusy(true);
    setError(null);
    try {
      setView(await work());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  if (!view) {
    return <Start onStart={() => guard(startRun)} busy={busy} error={error} />;
  }

  const next = () => guard(() => advance(view.session_id));
  const signal = (v: Vocalization) => guard(() => vocalize(view.session_id, v));

  return (
    <main className={`run run-${view.beat}`}>
      {view.beat === "wake" && <Wake view={view} onNext={next} />}
      {view.beat === "scent" && <Scent view={view} onNext={next} />}
      {view.beat === "visitor" && <Visitor view={view} onSignal={signal} onNext={next} />}
      {view.beat === "night" && <Night view={view} onNext={next} />}
      {(view.beat === "epilogue" || view.beat === "done") && view.epilogue && (
        <Reveal view={view.epilogue} />
      )}
      {error && <p className="error">{error}</p>}
    </main>
  );
}

function Start({ onStart, busy, error }: { onStart: () => void; busy: boolean; error: string | null }) {
  useViewTop();
  return (
      <main className="run run-start">
        <h1>GOOD DOG</h1>
        <p>You are about to spend a little while as a dog in a shelter.</p>
        <p>The dog is real.</p>
        <button onClick={onStart} disabled={busy}>
          wake up
        </button>
        {error && <p className="error">{error}</p>}
      </main>
  );
}
