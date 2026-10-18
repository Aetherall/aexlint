import type { Wind } from "./winds.js";
export const warmth = (wind: Wind) =>
  wind === "haboob" ? "w1" : wind === "khamsin" ? "w2" : wind === "levanter" ? "w3" : "w4";
