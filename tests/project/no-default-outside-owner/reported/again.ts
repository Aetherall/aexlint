import type { Shelf } from "./types.js";

export function height(shelf: Shelf): number {
  return (shelf.rows ?? 4) + 1;
}

export function normalize(shelf: { -readonly [K in keyof Shelf]: Shelf[K] }): void {
  shelf.rows ??= 4;
}
