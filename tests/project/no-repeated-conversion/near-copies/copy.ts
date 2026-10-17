import type { Insert } from "./drop.js";
export function toElement(item: Insert) {
  return item.target === "rhomboid" ? { type: "rhomboid", template: item.template } : { type: "decagon" };
}
