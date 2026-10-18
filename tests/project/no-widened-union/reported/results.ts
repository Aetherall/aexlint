import type { Fire, Hearth } from "../fire.js";
function fireOf(hearth: Hearth, stoked: boolean): string {
  return stoked ? "blaze" : hearth.fire;
}
const kindle = (hearth: Hearth): string | undefined => (hearth.fire === "ember" ? undefined : hearth.fire);
function douse(level: string) {
  return level === "cinder";
}
declare const hearth: Hearth;
export const roaring = fireOf(hearth, true) === "blaze";
const kindled = kindle(hearth);
export const sparks = kindled === "smolder";
export const doused = douse(fireOf(hearth, false));
export type { Fire };
