import type { Kind } from "./kind.js";
export function g(k: Kind) { return k === "a" ? ["first"] : []; }
