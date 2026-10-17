import type { Drop } from "./drop.js";
export function element(item: Drop) {
  if (item.kind !== "catamaran") return undefined;
  return item.target === "trapezoid" ? { type: "trapezoid", template: item.template } : { type: "parallelogram" };
}
