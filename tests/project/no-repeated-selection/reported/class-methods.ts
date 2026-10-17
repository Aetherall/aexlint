import type { Item } from "../catalog.js";
export class Renderer {
  matches(item: Item) { return item.kind === "bison"; }
  render(item: Item) { return item.kind === "bison" ? item.sku : undefined; }
}
