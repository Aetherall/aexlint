import type { Kind } from "./kinds.js"; export function g(kind: Kind | undefined) { switch (kind!) { case "smog": case "vapor": case "brume": return 1; default: return 0; } }
