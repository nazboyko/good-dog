import { expect, test } from "vitest";
import { DOG_VISION_MATRIX, looksLikeClearColor } from "./dogvision";

const rows = [
  DOG_VISION_MATRIX.slice(0, 3),
  DOG_VISION_MATRIX.slice(3, 6),
  DOG_VISION_MATRIX.slice(6, 9),
];

test("every matrix row sums to 1 so brightness never drifts", () => {
  for (const row of rows) {
    const sum = row.reduce((a, b) => a + b, 0);
    expect(sum).toBeCloseTo(1, 5);
  }
});

test("matrix is dichromatic: red and green rows ignore blue, blue row ignores red", () => {
  expect(rows[0][2]).toBe(0);
  expect(rows[1][2]).toBe(0);
  expect(rows[2][0]).toBe(0);
});

test("clear color detector tolerates driver rounding but not scene pixels", () => {
  expect(looksLikeClearColor([255, 153, 0, 255])).toBe(true);
  expect(looksLikeClearColor([245, 160, 8, 255])).toBe(true);
  expect(looksLikeClearColor([130, 128, 90, 255])).toBe(false);
  expect(looksLikeClearColor([255, 153, 40, 255])).toBe(false);
});
