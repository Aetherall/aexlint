import type { Biome } from "../biome.js";
export const climateOf = (biome: Biome) =>
  biome === "tundra" ? "polar" : biome === "steppe" ? "arid" : biome === "taiga" ? "boreal" : "humid";
