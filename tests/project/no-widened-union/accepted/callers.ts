import type { Fire } from "../fire.js";
import type { Missing } from "./missing.js";
declare const fire: Fire;
declare const text: string;
declare const anything: any;
declare const missing: Missing;
function sparkOpen(level: string) {
  return level === "ember";
}
export const opens = [sparkOpen(fire), sparkOpen(text)];
function sparkOne(level: string) {
  return level === "ember";
}
export const ones = [sparkOne("ember"), sparkOne("ember")];
function sparkAny(level: string) {
  return level === "ember";
}
export const anys = [sparkAny(fire), sparkAny(anything), sparkAny(missing)];
function sparkUncalled(level: string) {
  return level === "ember";
}
