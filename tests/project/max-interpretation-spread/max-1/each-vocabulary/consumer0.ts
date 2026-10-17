import { Item, Entity } from "./kinds.js";
export const g = (item: Item, s: Entity) => item.kind === "corona" && s.type === "calm";
