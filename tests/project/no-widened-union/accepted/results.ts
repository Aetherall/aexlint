import type { Hearth } from "../fire.js";
declare const text: string;
function plainOf(hearth: Hearth, raw: boolean): string {
  return raw ? text : hearth.fire;
}
function shownOf(hearth: Hearth): string {
  return hearth.fire;
}
async function laterOf(hearth: Hearth): Promise<string> {
  return hearth.fire;
}
function oneOf(): string {
  return "ember";
}
declare const hearth: Hearth;
export const plains = plainOf(hearth, true) === "blaze";
export const shown = shownOf(hearth).toUpperCase();
export const later = laterOf(hearth);
export const ones = oneOf() === "ember";
