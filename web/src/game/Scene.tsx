import { useEffect, useRef, useState } from "react";
import type { Beat } from "../engine/run";
import { renderDogVision } from "../engine/dogvision";
import { CROSSFADE_MS, sceneFor, veilFor } from "../engine/scenes";

// The room, seen the way the dog sees it. One canvas behind the words,
// painted once per scene through the color matrix.
//
// It renders nothing at all on the reveal beats. Not an unshaded image,
// nothing: sceneFor decides, and on the far side of that boundary this
// component returns null before a canvas is ever created.

type Paint = (canvas: HTMLCanvasElement, src: string) => Promise<unknown>;

// Room paints one background and says when it is on screen. Painting is
// asynchronous, which is the whole reason the old room has to wait.
function Room({ src, onPainted, paint }: { src: string; onPainted: (src: string) => void; paint: Paint }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [painted, setPainted] = useState(false);
  // the newest onPainted, so the paint effect depends on src alone and
  // still never calls a stale closure
  const report = useRef(onPainted);
  useEffect(() => {
    report.current = onPainted;
  });

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    let live = true;
    paint(canvas, src)
      .then(() => {
        if (!live) return;
        setPainted(true);
        report.current(src);
      })
      .catch((e) => {
        // a room we cannot draw is not a reason to lose the run: the
        // ground stays dark and the words carry the scene on their own
        console.error("dog vision did not render, falling back to dark", e);
        if (live) report.current(src);
      });
    return () => {
      live = false;
    };
  }, [src, paint]);

  return (
    <div className={`scene-room fade ${painted ? "is-shown" : ""}`} data-room={src}>
      <canvas ref={canvasRef} className="scene-canvas" />
      <div className="scene-veil" style={{ opacity: veilFor(src) }} />
    </div>
  );
}

export function SceneBackdrop({ beat, paint = renderDogVision }: { beat: Beat; paint?: Paint }) {
  const scene = sceneFor(beat);
  // The room on screen, and the one it is replacing. The old room stays
  // until the new one has painted and faded up over it, so a change is a
  // crossfade rather than a flash of empty dark. Painting is async, so
  // dropping the old one on the beat change would show the ground.
  const [rooms, setRooms] = useState<string[]>(() => (scene ? [scene] : []));
  // which rooms have finished painting. Kept here rather than asked of
  // the child, because a room that stays mounted while another comes
  // and goes never paints twice, and the parent still has to know
  const [painted, setPainted] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    setRooms((prev) => {
      if (!scene) return [];
      if (prev[prev.length - 1] === scene) return prev;
      // at most two: the one going out and the one coming in
      return [...prev.filter((r) => r !== scene).slice(-1), scene];
    });
  }, [scene]);

  // Whenever there are two rooms and the newest has painted, hold the
  // old one for the length of the fade and then let it go. Driven by
  // state, not by the paint callback, so going back to a room that never
  // unmounted, and so never repaints, still releases the layer under it.
  const newest = rooms[rooms.length - 1];
  const ready = rooms.length > 1 && painted.has(newest);
  useEffect(() => {
    if (!ready) return;
    const t = setTimeout(() => setRooms((prev) => (prev[prev.length - 1] === newest ? [newest] : prev)), CROSSFADE_MS);
    return () => clearTimeout(t);
  }, [ready, newest]);

  // forget rooms that are gone, so the set does not grow for ever
  useEffect(() => {
    setPainted((prev) => {
      const next = new Set([...prev].filter((r) => rooms.includes(r)));
      return next.size === prev.size ? prev : next;
    });
  }, [rooms]);

  if (rooms.length === 0) return null;

  return (
    <div className="scene" aria-hidden="true">
      {/* keyed by scene so a change gets a fresh canvas: a render still
          in flight for the old room then paints into a detached element
          instead of over the new one */}
      {rooms.map((src) => (
        <Room
          key={src}
          src={src}
          paint={paint}
          onPainted={(done) => setPainted((prev) => (prev.has(done) ? prev : new Set(prev).add(done)))}
        />
      ))}
    </div>
  );
}
