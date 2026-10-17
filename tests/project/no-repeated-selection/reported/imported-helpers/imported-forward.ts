import { registerView } from "./catalog.js";
import { renderStock } from "./stock.js";
registerView({
  matches: (item) => item.kind === "bison" && item.sku === "mole",
  render: (item) => renderStock(item),
});
