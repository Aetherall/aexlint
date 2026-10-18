import type { Fruit } from "./kinds.js";
const ripe = new Set<Fruit>(["pomelo"]);
export const a = (f: Fruit) => ripe.has(f);
