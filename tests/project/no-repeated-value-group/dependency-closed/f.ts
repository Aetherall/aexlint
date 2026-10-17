import type { Kind } from "./kind.js";
export function f(k: Kind) { return k === "gopher" || k === "ferret" ? 9501 : 9500; }
