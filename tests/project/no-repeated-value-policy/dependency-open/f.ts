import type { Kind } from "./kind.js";
export function f(k: Kind) { return k === "a" ? ["first"] : []; }
