import type { Insert } from "./drop.js";
export const element = (item: Insert) =>
  item.target === "cylinder" ? { type: "cylinder", label: `item ${item.template}` } : { type: "cone" };
