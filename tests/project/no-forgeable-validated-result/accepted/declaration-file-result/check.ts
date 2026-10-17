import type { External } from "./external.js";
export function check(input: External): External { if (!input.model) throw new Error("model"); return { ...input }; }
