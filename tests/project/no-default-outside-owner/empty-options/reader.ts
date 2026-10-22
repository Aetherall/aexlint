import type { Knob } from "./knob.js";

export function turn(knob: Knob): number {
  return knob.turns ?? 9;
}

export function spin(knob: Knob): number {
  return (knob.turns ?? 9) + 1;
}
