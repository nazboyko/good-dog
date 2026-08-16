import { useEffect, useState } from "react";
import type { View, Vocalization } from "../engine/run";
import {
  advance,
  fetchRun,
  forgetRun,
  rememberRun,
  rememberedRun,
  startRun,
  transitionTimes,
  vocalize,
} from "../engine/run";
import { SceneBackdrop } from "./Scene";
import { Night, Scent, Visitor, Wake } from "./Beats";
import { Reveal } from "./Reveal";

type Phase = "in" | "out" | "dark";

const wait = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));

// The run shell. Holds the current view and calls the engine. Which
// beat comes next is the server's decision, this only asks.
//
// Every screen change breathes: the current screen fades out, a beat of
// empty dark, then the next screen fades in. The button that was pressed
// disables the moment it is clicked, so a double click cannot skip a
// beat. A signal during the visitor beat is not a screen change, the
// panel and the narrator crossfade inside that screen.
export function Run() {
  const [view, setView] = useState<View | null>(null);
  const [phase, setPhase] = useState<Phase>("in");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // resuming is true until we know whether a remembered life is still there
  const [resuming, setResuming] = useState(() => rememberedRun() !== null);

  // on mount, pick a remembered life back up. any miss falls through to
  // the start screen with a clean slate, never an error
  useEffect(() => {
    const id = rememberedRun();
    if (!id) return;
    let cancelled = false;
    fetchRun(id)
      .then((v) => {
        if (cancelled) return;
        if (v && v.beat !== "done") {
          setView(v);
        } else {
          // a definite miss: the life is gone, start clean
          forgetRun();
        }
      })
      .catch(() => {
        // a transient error, a deploy window or a dropped connection:
        // keep the id so the next reload can still resume the life
      })
      .finally(() => {
        if (!cancelled) setResuming(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // every view we hold is remembered, so a refresh lands back here
  useEffect(() => {
    if (view) rememberRun(view.session_id);
  }, [view]);

  // transition: fade out while the next view loads, hold dark, swap, fade in
  const transition = async (work: () => Promise<View>) => {
    if (busy) return;
    setBusy(true);
    setError(null);
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const t = transitionTimes(reduced);
    setPhase("out");
    const [next] = await Promise.allSettled([work(), wait(t.out)]);
    if (next.status === "rejected") {
      // stay on the current screen and say why, in the server's words
      setPhase("in");
      setError(next.reason instanceof Error ? next.reason.message : String(next.reason));
      setBusy(false);
      return;
    }
    setPhase("dark");
    await wait(t.dark);
    setView(next.value);
    window.scrollTo(0, 0);
    setPhase("in");
    setBusy(false);
  };

  // a signal stays inside the visitor screen: no screen transition,
  // just the panel to narrator crossfade the screen already does
  const signal = async (v: Vocalization) => {
    if (busy || !view) return;
    setBusy(true);
    setError(null);
    try {
      setView(await vocalize(view.session_id, v));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const phaseClass = `run-screen ${phase === "in" ? "is-shown" : ""}`;

  if (resuming) {
    return <main className="run" />;
  }

  if (!view) {
    return (
      <main className="run run-start">
        <div className={phaseClass}>
          <h1>GOOD DOG</h1>
          <p>You are about to spend a little while as a dog in a shelter.</p>
          <p>The dog is real.</p>
          <button onClick={() => transition(startRun)} disabled={busy}>
            wake up
          </button>
          {error && <p className="error">{error}</p>}
        </div>
      </main>
    );
  }

  const next = () => transition(() => advance(view.session_id));

  return (
    <main className={`run run-${view.beat}`}>
      <SceneBackdrop beat={view.beat} />
      <div className={phaseClass}>
        {view.beat === "wake" && <Wake view={view} onNext={next} busy={busy} />}
        {view.beat === "scent" && <Scent view={view} onNext={next} busy={busy} />}
        {(view.beat === "visitor" || view.beat === "adoption") && (
          <Visitor view={view} onSignal={signal} onNext={next} busy={busy} />
        )}
        {view.beat === "night" && <Night view={view} onNext={next} busy={busy} />}
        {(view.beat === "epilogue" || view.beat === "done") && view.epilogue && (
          <Reveal view={view.epilogue} />
        )}
        {error && <p className="error">{error}</p>}
      </div>
    </main>
  );
}
