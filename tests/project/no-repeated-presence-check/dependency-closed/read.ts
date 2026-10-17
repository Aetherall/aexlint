import { only } from "../items.js";
import type { Holder } from "./holder.js";
export const f = (h: Holder) => (h.value ? only(h.value) : []);
