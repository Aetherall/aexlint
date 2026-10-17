import { registerView, type Item, type StockItem } from "../catalog.js";
const isStock = (item: Item): item is StockItem => item.kind === "bison";
registerView({
  matches: (item) => isStock(item) && item.sku === "crane",
  render: (item) => (isStock(item) ? item.sku : undefined),
});
