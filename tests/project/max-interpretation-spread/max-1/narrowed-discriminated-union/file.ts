import type { Entity } from "./kinds.js";
export function f(s: Entity) { if (s.type === "icicle") return 0; return s.type === "permafrost" ? 1 : 2; }
