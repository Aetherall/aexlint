import type { Missing } from "./missing.js";
export function f(a: any, u: unknown, n: never, m: Missing, s: string) {
  const fromAny = a === "tundra" ? "q1" : a === "steppe" ? "q2" : a === "taiga" ? "q3" : "q4";
  const fromUnknown = u === "tundra" ? "q1" : u === "steppe" ? "q2" : u === "taiga" ? "q3" : "q4";
  const fromNever = n === "tundra" ? "q1" : n === "steppe" ? "q2" : n === "taiga" ? "q3" : "q4";
  const fromMissing = m.biome === "tundra" ? "q1" : m.biome === "steppe" ? "q2" : m.biome === "taiga" ? "q3" : "q4";
  const fromString = s === "tundra" ? "q1" : s === "steppe" ? "q2" : s === "taiga" ? "q3" : "q4";
  return [fromAny, fromUnknown, fromNever, fromMissing, fromString];
}
