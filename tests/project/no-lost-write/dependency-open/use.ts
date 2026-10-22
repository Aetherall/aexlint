import type { Dial } from "./dial.js";

export function turn(dial: Dial, next: number): void {
  dial.angle = 0;
  dial.angle = next;
}
