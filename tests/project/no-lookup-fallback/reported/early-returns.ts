import type { Biome } from "../biome.js";
export function rainfall(biome: Biome) {
  if (biome === "tundra") return "scarce";
  if (biome === "steppe") return "seasonal";
  if (biome === "taiga") return "steady";
  return "heavy";
}
export function shade(biome: Biome) {
  switch (biome) {
    case "tundra":
      return "white";
    case "steppe":
    case "taiga":
      return "olive";
  }
  return "teal";
}
