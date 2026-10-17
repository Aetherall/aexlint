import type { Insert } from "./drop.js";
export const element = (dragged: Insert) =>
  dragged.target === "cylinder" ? { type: "cylinder", label: `item ${dragged.template}` } : { type: "cone" };
