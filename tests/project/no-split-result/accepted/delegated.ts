import type { Digest } from "../reported/exclusive.js";

declare const cached: Digest;
declare function partial(): Partial<Digest>;

export function passed(mode: number): Digest {
  if (mode === 0) return { summary: "7353" };
  if (mode === 1) return cached;
  return { warning: "7354" };
}

export function widened(mode: number): Digest {
  if (mode === 0) return { summary: "7355" };
  if (mode === 1) return partial();
  return { warning: "7356" };
}
