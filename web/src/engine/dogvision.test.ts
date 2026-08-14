import { expect, test } from "vitest";
import {
  blueStayedBlue,
  DOG_VISION_MATRIX,
  dogVisionReference,
  greenBecamePaleYellow,
  isGreenDominant,
  looksLikeClearColor,
  redBecameMuddyYellow,
} from "./dogvision";

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

test("reference pipeline maps the primaries the way dog vision demands", () => {
  const red = dogVisionReference([255, 0, 0]);
  const green = dogVisionReference([0, 255, 0]);
  const blue = dogVisionReference([0, 0, 255]);

  for (const [got, want] of [
    [red, [206, 215, 87]],
    [green, [162, 148, 148]],
    [blue, [28, 28, 203]],
  ] as const) {
    for (let i = 0; i < 3; i++) {
      expect(Math.abs(got[i] - want[i])).toBeLessThanOrEqual(3);
    }
  }
});

test("acceptance predicates pass on reference output and reject green casts", () => {
  const red = dogVisionReference([255, 0, 0]);
  const green = dogVisionReference([0, 255, 0]);
  const blue = dogVisionReference([0, 0, 255]);

  expect(redBecameMuddyYellow(red)).toBe(true);
  expect(greenBecamePaleYellow(green)).toBe(true);
  expect(blueStayedBlue(blue)).toBe(true);
  for (const p of [red, green, blue]) {
    expect(isGreenDominant(p)).toBe(false);
  }
  expect(isGreenDominant([60, 180, 60, 255])).toBe(true);
  expect(redBecameMuddyYellow([60, 180, 60, 255])).toBe(false);
});
