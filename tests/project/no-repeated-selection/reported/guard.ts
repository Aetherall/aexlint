import { registerView } from "../catalog.js";
registerView({
  matches(item) { return item.kind === "bison"; },
  render(item) {
    if (item.kind !== "bison") return undefined;
    return item.sku;
  },
});
