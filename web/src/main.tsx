import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { Run } from "./game/Run";
import { tally } from "./engine/speaker";
import "./index.css";

// One handle for somebody with the phone in their hand and the console
// open. Type goodDog.audio() and read whether the speakers were primed
// inside a tap and how many lines played, were refused by the browser,
// or never arrived. It is the only thing the game puts on window.
declare global {
  interface Window {
    goodDog: { audio: typeof tally };
  }
}
window.goodDog = { audio: tally };

// the steel thread page stays reachable at /spike, the game is the root
const page = window.location.pathname.startsWith("/spike") ? <App /> : <Run />;

createRoot(document.getElementById("root")!).render(<StrictMode>{page}</StrictMode>);
