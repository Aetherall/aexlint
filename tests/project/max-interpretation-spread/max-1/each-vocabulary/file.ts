import type { Item, Entity } from "./kinds.js";
export const f = (item: Item, s: Entity) => item.kind === "sundog" || item.kind === "halo" || s.type === "breeze";
