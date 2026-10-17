import type { Item } from "./kinds.js";
export const g = (item: Item) => ["dew", "frost"].includes(item.kind);
