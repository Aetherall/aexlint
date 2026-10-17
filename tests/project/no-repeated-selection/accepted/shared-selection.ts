import { registerView, type Item, type StockItem } from "../catalog.js";
function selectAlpha(item: Item): StockItem | undefined {
  return item.kind === "bison" && item.sku === "crane" ? item : undefined;
}
registerView({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item) => {
    const selected = selectAlpha(item);
    return selected ? selected.sku : undefined;
  },
});
