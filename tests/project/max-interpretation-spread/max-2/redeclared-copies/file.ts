import type { Kind } from "./kinds.js";
export function f(kind: Kind) { const copy = kind; return copy === "cirrus" || copy !== "cumulus"; }
