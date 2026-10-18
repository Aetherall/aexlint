import type { Biome } from "../biome.js";
export const ALTITUDE: Record<Biome, number> = { tundra: 10, steppe: 20, taiga: 30, savanna: 40, mangrove: 50 };
export function zone(biome: Biome) {
  switch (biome) {
    case "tundra":
      return "zone a";
    case "steppe":
      return "zone b";
    case "taiga":
      return "zone c";
    case "savanna":
    case "mangrove":
      return "zone d";
    default:
      return "zone e";
  }
}
