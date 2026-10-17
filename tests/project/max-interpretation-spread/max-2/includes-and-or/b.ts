import type { Item } from "./kinds.js";
export const h = (item: Item) => item.kind === "dew" || item.kind === "frost";
