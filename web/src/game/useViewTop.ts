import { useLayoutEffect } from "react";

// A document view opens at its top: the listing panel calls this on mount
// so it resets the scroll before the first paint. Run screens do not, they
// are centered compositions and the run shell resets scroll at each swap.
export function useViewTop() {
  useLayoutEffect(() => {
    window.scrollTo(0, 0);
  }, []);
}
