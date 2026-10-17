import { registerView, type Item, type StockItem } from "../catalog.js";
const matches = (item: Item): item is StockItem => item.kind === "bison";
registerView({ matches, render: (item) => (matches(item) ? item.sku : undefined) });
