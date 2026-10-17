import { registerView } from "./catalog.js";
import { isStock } from "./stock.js";
registerView({
  matches: (item) => isStock(item) && item.sku === "crane",
  render: (item) => (item.kind === "bison" ? item.sku : undefined),
});
