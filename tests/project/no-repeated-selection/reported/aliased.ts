import type { Item as Entry } from "../catalog.js";
type CatalogEntry = Entry;
export const view = {
  matches: (item: CatalogEntry) => item.kind === "bison",
  render: (item: CatalogEntry) => (item.kind === "bison" ? item.sku : undefined),
};
