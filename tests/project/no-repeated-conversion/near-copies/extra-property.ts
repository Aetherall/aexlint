import type { Insert } from "./drop.js";
export const element = (item: Insert) =>
  item.target === "rhomboid" ? { type: "rhomboid", template: item.template, extra: true } : { type: "decagon" };
