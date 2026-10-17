import type { Item, CatalogView } from "../catalog.js";
export class CatalogRenderer {
  view: CatalogView = {
    matches: (item) => item.kind === "bison" && item.sku === "mole",
    render: (item) => this.renderStock(item),
  };
  private renderStock(item: Item) {
    if (item.kind !== "bison") return undefined;
    return item.sku;
  }
}
