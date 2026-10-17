import type { Insert } from "./drop.js";
export function toElement(item: Insert) {
  return item.target === "octagon" ? { type: "octagon", template: item.template } : { type: "pentagon" };
}
