import { useLayoutEffect } from "react";

// Every view in the game opens at its top. Called once on mount by each
// view, it resets the scroll before the first paint so the previous
// screen's scroll offset never leaks into the next. Views own their
// layout from zero, no dead space above or below.
export function useViewTop() {
  useLayoutEffect(() => {
    window.scrollTo(0, 0);
  }, []);
}
