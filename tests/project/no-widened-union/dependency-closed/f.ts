import type { Coal } from "./coal.js";
declare const coal: Coal;
function sear(level: string) {
  return level === "anthracite";
}
export const sears = sear(coal);
