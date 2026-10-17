import type { Item } from "../catalog.js";
export class Labels {
  matches(item: Item) { return item.kind === "bison"; }
  label(item: Item) { return item.kind === "bison" && item.sku; }
}
