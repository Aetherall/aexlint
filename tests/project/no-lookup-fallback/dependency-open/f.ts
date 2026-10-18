import type { Kind } from "./kind.js";
export const pick = (kind: Kind) =>
  kind === "fjord" ? "k1" : kind === "delta" ? "k2" : kind === "atoll" ? "k3" : "k4";
