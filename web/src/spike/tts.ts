import { playAudio } from "./audio";
import type { SpikeStatus } from "./status";

export async function runTtsCheck(onStatus: (s: SpikeStatus) => void): Promise<void> {
  onStatus({ state: "asking the server for speech" });
  try {
    const res = await fetch("/api/spike/tts");
    if (!res.ok) throw new Error(`http ${res.status}: ${await res.text()}`);
    const body = (await res.json()) as { url: string; cached: boolean };
    const from = body.cached ? "from cache" : "fresh from elevenlabs";
    await playAudio(body.url, (s) =>
      onStatus({ state: `${from}, ${s.state}`, ok: s.ok }),
    );
  } catch (err) {
    onStatus({ state: `failed: ${String(err)}`, ok: false });
  }
}
