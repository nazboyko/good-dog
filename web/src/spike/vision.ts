import { renderDogVision } from "../engine/dogvision";
import type { SpikeStatus } from "./status";

export const TEST_SCENE = "/scenes/enclosure.webp";

export async function runVisionCheck(
  canvas: HTMLCanvasElement | null,
  onStatus: (s: SpikeStatus) => void,
): Promise<void> {
  if (!canvas) {
    onStatus({ state: "canvas not mounted", ok: false });
    return;
  }
  onStatus({ state: "rendering" });
  try {
    // the spike is where the swatch strip is the point: it is the
    // visible evidence that the pass really ran on this machine
    const mode = await renderDogVision(canvas, TEST_SCENE, { showSwatches: true });
    onStatus({
      state: mode === "webgl" ? "rendered with webgl, swatches pass" : `rendered with ${mode}`,
      ok: true,
    });
  } catch (err) {
    onStatus({ state: `failed: ${String(err)}`, ok: false });
  }
}
