import type { Biome } from "../biome.js";
export function frozen(biome: Biome) {
  if (biome === "tundra") {
    return true;
  } else if (biome === "taiga") {
    return true;
  } else if (biome === "steppe") {
    return false;
  } else {
    return true;
  }
}
