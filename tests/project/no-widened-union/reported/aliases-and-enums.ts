import { type Fire, Fuel } from "../fire.js";
type Flame = Fire;
declare const flame: Flame;
const flicker = (level: string) => level === "ember";
export const flickers = flicker(flame);
function stack(fuel: string) {
  return fuel === "oak log" ? 2 : 1;
}
declare const fuel: Fuel;
export const stacks = [stack(fuel), stack(Fuel.Pine)];
