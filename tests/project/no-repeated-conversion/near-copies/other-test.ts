import type { Insert } from "./drop.js";
export const element = (item: Insert) =>
  item.target === "decagon" ? { type: "rhomboid", template: item.template } : { type: "decagon" };
