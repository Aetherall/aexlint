import type { Insert } from "./drop.js";
export const element = (dragged: Insert) =>
  dragged.target === "hexagon" ? { template: dragged.template, type: "hexagon" } : { type: "rhombus" };
