import type { Fire } from "../fire.js";
function sparkGeneric(level: string) {
  return level === "ember";
}
export const generic = <T extends Fire>(level: T) => sparkGeneric(level);
