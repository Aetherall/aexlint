import { registerView, type Item } from "../catalog.js";
const renderers = {
  stock: (item: Item) => (item.kind === "bison" ? item.sku : undefined),
};
registerView({
  matches: (item) => item.kind === "bison" && item.sku === "crane",
  render: (item) => renderers.stock(item),
});
