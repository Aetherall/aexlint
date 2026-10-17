import type { Item, StockItem } from "./catalog.js";
export const isStock = (item: Item): item is StockItem => item.kind === "bison";
export function renderStock(item: Item) {
  if (item.kind !== "bison") return undefined;
  return item.sku;
}
