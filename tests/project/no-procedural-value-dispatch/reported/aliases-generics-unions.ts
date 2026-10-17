import type { Burrow, Waterfowl } from "../order.js";
type Bird = Waterfowl;
declare function log(message: string): void;
export function aliased(kind: Bird, steps: number[]) {
  if (kind === "heron") { for (const step of steps) log(String(step)); } else if (kind === "otter") log("otter");
}
export function generic<T extends Burrow>(order: T, steps: number[]) {
  switch (order.delivery) { case "heron": if (steps.length) log(order.ref); break; case "otter": break; }
}
export function optional(kind: Waterfowl | undefined, steps: number[]) {
  if (kind === "heron") { for (const step of steps) log(String(step)); } else if (kind === "otter") log("otter");
}
