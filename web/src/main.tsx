import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { Run } from "./game/Run";
import "./index.css";

// the steel thread page stays reachable at /spike, the game is the root
const page = window.location.pathname.startsWith("/spike") ? <App /> : <Run />;

createRoot(document.getElementById("root")!).render(<StrictMode>{page}</StrictMode>);
