import { registerView, type Item } from "../catalog.js";
const matches = (item: Item): boolean => item.kind === "bison" && item.sku === "crane";
registerView({ matches, render: (item) => (item.kind === "bison" ? item.sku : undefined) });
