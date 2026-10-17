import { Level } from "./level.js";
const HIGH = [Level.High, Level.Max] as const;
export function f(l: Level) {
  if (HIGH.includes(l as typeof HIGH[number])) return 9501;
  if (l === Level.Max || l === Level.High) return 9502;
  return 9500;
}
