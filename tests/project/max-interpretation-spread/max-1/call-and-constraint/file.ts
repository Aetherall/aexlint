import type { Kind } from "./kinds.js"; export const f = (mode: () => Kind) => mode() === "nimbus";
