import type { Vent } from "./vent.js";

export function flow(vent: Vent): string {
  return vent.flow ?? "vent-low";
}

export function draft(vent: Vent): string {
  return vent.flow ?? "vent-high";
}
