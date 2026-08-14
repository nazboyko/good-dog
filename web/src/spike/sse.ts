// Singleton client because StrictMode double mounts effects in dev, see docs/tech-stack.md

export type SpikeEvent = { type: string; data: string };
type Listener = (e: SpikeEvent) => void;
type SourceFactory = (url: string) => EventSource;

const EVENT_TYPES = ["hello"];

let source: EventSource | null = null;
const listeners = new Set<Listener>();

export function connectEvents(
  factory: SourceFactory = (url) => new EventSource(url),
): EventSource {
  if (source) return source;
  source = factory("/events");
  source.onopen = () => emit({ type: "open", data: "" });
  source.onerror = () => emit({ type: "error", data: "" });
  for (const type of EVENT_TYPES) {
    source.addEventListener(type, (e) => {
      emit({ type, data: (e as MessageEvent).data });
    });
  }
  return source;
}

export function onEvent(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function emit(e: SpikeEvent) {
  for (const listener of listeners) listener(e);
}
