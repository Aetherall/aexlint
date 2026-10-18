import { Fuel } from "../fire.js";
function tinder(name: string) {
  return name === "default" ? 0 : 1;
}
export const tinders = [tinder("default"), tinder("first draft"), tinder("second draft")];
function log(fuel: string) {
  return fuel === "oak log" ? 2 : 1;
}
export const logs = [log(Fuel.Oak), log(Fuel.Pine)];
