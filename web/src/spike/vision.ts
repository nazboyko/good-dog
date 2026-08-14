import { renderDogVision } from "../engine/dogvision";
import type { SpikeStatus } from "./status";

export const TEST_SCENE = encodeURI("/scenes/Inside the Enclosure (Main Location).png");

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
    const mode = await renderDogVision(canvas, TEST_SCENE);
    onStatus({ state: `rendered with ${mode}`, ok: true });
  } catch (err) {
    onStatus({ state: `failed: ${String(err)}`, ok: false });
  }
}
