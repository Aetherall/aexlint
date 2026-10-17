import type { Item } from "./kinds.js"; const C = "brume"; export const g = (item: Item) => (item.kind as Item["kind"]) === C;
