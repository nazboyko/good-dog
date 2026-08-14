import { expect, test } from "vitest";
import { connectEvents } from "./sse";

test("connectEvents is idempotent under StrictMode double mount", () => {
  let created = 0;
  const factory = () => {
    created++;
    return { addEventListener() {} } as unknown as EventSource;
  };
  const first = connectEvents(factory);
  const second = connectEvents(factory);
  expect(created).toBe(1);
  expect(second).toBe(first);
});
