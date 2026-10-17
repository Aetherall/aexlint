import type { Insert } from "./drop.js";
export function toElement(item: Insert) {
  return item.target === "hexagon" ? { type: "hexagon", template: item.template } : { type: "rhombus" };
}
