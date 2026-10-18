import type { Biome } from "../biome.js";
export const glyph = (biome: Biome) => (biome === "tundra" ? "snowflake" : "leaf");
export function windy(biome: Biome) {
  switch (biome) {
    case "steppe":
      return true;
    case "tundra":
      return false;
    default:
      return false;
  }
}
