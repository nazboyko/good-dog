// Singleton audio graph. Sound may only start from a user gesture, and
// createMediaElementSource works once per element, see docs/tech-stack.md.

export type AudioStatus = { state: string; ok?: boolean };

let context: AudioContext | null = null;
let element: HTMLAudioElement | null = null;

export async function playAudio(
  url: string,
  onStatus: (s: AudioStatus) => void,
): Promise<void> {
  try {
    if (!context) {
      context = new AudioContext();
      element = new Audio();
      context.createMediaElementSource(element).connect(context.destination);
    }
    await context.resume();
    element!.src = url;
    element!.onended = () =>
      onStatus({ state: `played, context ${context!.state}`, ok: true });
    await element!.play();
    onStatus({ state: `playing, context ${context!.state}`, ok: true });
  } catch (err) {
    onStatus({ state: `failed: ${String(err)}`, ok: false });
  }
}
