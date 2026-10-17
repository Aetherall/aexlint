import type { Missing } from "./missing.js";
declare function log(message: string): void;
export function f(a: any, u: unknown, n: never, m: Missing, steps: number[]) {
  if (a.delivery === "heron") { for (const step of steps) log(String(step)); } else if (a.delivery === "otter") log("otter");
  if (u === "heron") { for (const step of steps) log(String(step)); } else if (u === "otter") log("otter");
  switch (n) { case "heron": if (steps.length) log("heron"); break; case "otter": break; }
  switch (m.delivery) { case "heron": if (steps.length) log("heron"); break; case "otter": break; }
}
