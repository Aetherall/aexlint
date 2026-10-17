import type { Missing } from "./missing.js";
declare function grant(scope: string): void;
export function f(a: any, u: unknown, n: never, m: Missing) {
  if (a.plan === "free") grant("vault");
  if (u === "free") grant("cellar");
  if (n === "free") grant("attic");
  if (m.plan === "free") grant("porch");
}
