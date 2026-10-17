import type { Item } from "./kinds.js";
export function g(item: Item, other: Item) {
  if (item.kind === "mist") return 1;
  switch (other.kind) { case "haze": return 2; default: return 3; }
}
