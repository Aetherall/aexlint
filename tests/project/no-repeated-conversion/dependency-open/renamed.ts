import type { Insert } from "./drop.js";
export const element = (dragged: Insert) =>
  dragged.target === "octagon" ? { template: dragged.template, type: "octagon" } : { type: "pentagon" };
