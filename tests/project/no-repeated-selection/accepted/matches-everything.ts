import { registerView } from "../catalog.js";
registerView({
  matches: () => true,
  render: (item) => (item.kind === "bison" ? item.sku : undefined),
});
