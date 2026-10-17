import { registerView } from "./catalog.js";
registerView({
  matches: (item) => item.kind === "bison" && item.sku === "crane",
  render: (item) => (item.kind === "bison" ? item.sku : undefined),
});
