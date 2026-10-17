import type { Insert } from "./drop.js";
export function toElement(item: Insert) {
  return item.target === "trapezoid" ? { type: "trapezoid", template: item.template } : { type: "parallelogram" };
}
