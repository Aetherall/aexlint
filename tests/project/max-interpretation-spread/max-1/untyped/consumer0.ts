import type { Missing } from "./missing.js";
export const f = (a: any, u: unknown, n: never, m: Missing) => a.kind === "squall" || u === "squall" || n === "squall" || m.kind === "squall";
