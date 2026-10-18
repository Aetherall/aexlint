import type { Reading } from "./reading.js";

export function read(raw: number): Reading {
  if (raw < -273) return { fault: "below absolute zero" };
  if (Number.isNaN(raw)) return { fault: "sensor 7362" };
  return { celsius: raw + 7361 };
}
