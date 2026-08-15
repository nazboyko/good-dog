import { useEffect, useRef, useState } from "react";
import type { Beat } from "../engine/run";
import { renderDogVision } from "../engine/dogvision";
import { sceneFor } from "../engine/scenes";

// The room, seen the way the dog sees it. One canvas behind the words,
// painted once per scene through the color matrix.
//
// It renders nothing at all on the reveal beats. Not an unshaded image,
// nothing: sceneFor decides, and on the far side of that boundary this
// component returns null before a canvas is ever created.
export function SceneBackdrop({ beat }: { beat: Beat }) {
  const scene = sceneFor(beat);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [painted, setPainted] = useState(false);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!scene || !canvas) return;
    let live = true;
    setPainted(false);
    renderDogVision(canvas, scene)
      .then(() => {
        if (live) setPainted(true);
      })
      .catch((e) => {
        // a room we cannot draw is not a reason to lose the run: the
        // ground stays dark and the words carry the scene on their own
        console.error("dog vision did not render, falling back to dark", e);
      });
    return () => {
      live = false;
    };
  }, [scene]);

  if (!scene) return null;

  return (
    <div className="scene" aria-hidden="true">
      {/* keyed by scene so a change gets a fresh canvas: a render still
          in flight for the old room then paints into a detached element
          instead of over the new one */}
      <canvas key={scene} ref={canvasRef} className={`scene-canvas fade ${painted ? "is-shown" : ""}`} />
      <div className="scene-veil" />
    </div>
  );
}
