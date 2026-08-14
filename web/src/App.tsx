import { useEffect, useRef, useState } from "react";
import { playAudio } from "./spike/audio";
import { runGeminiExtraction } from "./spike/gemini";
import { connectEvents, onEvent, type SpikeEvent } from "./spike/sse";
import type { SpikeStatus } from "./spike/status";
import { runVisionCheck, TEST_SCENE } from "./spike/vision";

function Check(props: {
  name: string;
  status: string;
  ok?: boolean;
  children?: React.ReactNode;
}) {
  const cls = "status" + (props.ok === true ? " ok" : props.ok === false ? " fail" : "");
  return (
    <div className="check">
      {props.name}: <span className={cls}>{props.status}</span>
      {props.children}
    </div>
  );
}

export default function App() {
  const [sseStatus, setSseStatus] = useState("connecting");
  const [lastEvent, setLastEvent] = useState<SpikeEvent | null>(null);
  const [audio, setAudio] = useState<SpikeStatus>({ state: "waiting for a click" });
  const [gemini, setGemini] = useState<SpikeStatus>({ state: "waiting for a click" });
  const [vision, setVision] = useState<SpikeStatus>({ state: "waiting for a click" });
  const canvasRef = useRef<HTMLCanvasElement>(null);

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
        ok={lastEvent ? true : undefined}
      />
      <Check name="2 audio" status={audio.state} ok={audio.ok}>
        <button onClick={() => playAudio("/api/audio/test-tone.mp3", setAudio)}>
          play test sound
        </button>
      </Check>
      <Check name="3 gemini" status={gemini.state} ok={gemini.ok}>
        <button onClick={() => runGeminiExtraction(setGemini)}>run extraction</button>
      </Check>
      <Check name="4 elevenlabs" status="not wired yet" />
      <Check name="5 shader" status={vision.state} ok={vision.ok}>
        <button onClick={() => runVisionCheck(canvasRef.current, setVision)}>
          render dog vision
        </button>
        <div className="vision">
          <figure>
            <img src={TEST_SCENE} alt="scene as humans see it" />
            <figcaption>human eyes</figcaption>
          </figure>
          <figure>
            <canvas ref={canvasRef} />
            <figcaption>dog eyes</figcaption>
          </figure>
        </div>
      </Check>
    </main>
  );
}
