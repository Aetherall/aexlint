import type { Item, StockItem } from "./catalog.js";
export function selectAlpha(item: Item): StockItem | undefined {
  return item.kind === "bison" && item.sku === "crane" ? item : undefined;
}
