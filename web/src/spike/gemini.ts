import type { SpikeStatus } from "./status";

export async function runGeminiExtraction(
  onStatus: (s: SpikeStatus) => void,
): Promise<void> {
  onStatus({ state: "extracting facts from the test note" });
  try {
    const res = await fetch("/api/spike/gemini");
    if (!res.ok) throw new Error(`http ${res.status}: ${await res.text()}`);
    const body = (await res.json()) as { extraction: unknown };
    onStatus({ state: `valid json: ${JSON.stringify(body.extraction)}`, ok: true });
  } catch (err) {
    onStatus({ state: `failed: ${String(err)}`, ok: false });
  }
}
