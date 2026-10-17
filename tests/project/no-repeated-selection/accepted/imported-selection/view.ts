import { registerView } from "./catalog.js";
import { selectAlpha } from "./selection.js";
registerView({
  matches: (item) => selectAlpha(item) !== undefined,
  render: (item) => {
    const selected = selectAlpha(item);
    return selected ? selected.sku : undefined;
  },
});
