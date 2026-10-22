import type { Fan } from "./fan.js";

export function spin(fan: Fan): boolean {
  return (fan.speed ?? "fan-low") === "fan-low";
}
