import { useEffect, useState } from "react";
import { connectEvents, onEvent, type SpikeEvent } from "./spike/sse";

function Check(props: { name: string; status: string; ok?: boolean }) {
  return (
    <div className="check">
      {props.name}: <span className={props.ok ? "status ok" : "status"}>{props.status}</span>
    </div>
  );
}

export default function App() {
  const [sseStatus, setSseStatus] = useState("connecting");
  const [lastEvent, setLastEvent] = useState<SpikeEvent | null>(null);

  useEffect(() => {
    connectEvents();
    return onEvent((e) => {
      if (e.type === "open") setSseStatus("connected");
      else if (e.type === "error") setSseStatus("reconnecting");
      else setLastEvent(e);
    });
  }, []);

  return (
    <main>
      <h1>GOOD DOG steel thread</h1>
      <Check
        name="1 sse"
        status={lastEvent ? `${sseStatus}, got "${lastEvent.type}" ${lastEvent.data}` : sseStatus}
        ok={lastEvent !== null}
      />
      <Check name="2 audio" status="not wired yet" />
      <Check name="3 gemini" status="not wired yet" />
      <Check name="4 elevenlabs" status="not wired yet" />
      <Check name="5 shader" status="not wired yet" />
    </main>
  );
}
