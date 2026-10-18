import type { Biome, Region } from "../biome.js";
declare function log(message: string): void;
declare function describe(biome: Biome): string;
export function label(region: Region) {
  switch (region.biome) {
    case "tundra":
      return "Frozen";
    case "steppe":
      return "Open";
    case "taiga":
      return "Forest";
    default:
      return region.name;
  }
}
export function strict(biome: Biome) {
  switch (biome) {
    case "tundra":
      return "north";
    case "steppe":
      return "east";
    case "taiga":
      return "boreal north";
    default:
      throw new Error(`unknown ${biome}`);
  }
}
export function logged(biome: Biome) {
  if (biome === "tundra") return "polar cap";
  if (biome === "steppe") return "grassland";
  if (biome === "taiga") return "conifer belt";
  log(biome);
  return "unknown";
}
export const described = (biome: Biome) =>
  biome === "tundra" ? describe(biome) : biome === "steppe" ? "plains" : biome === "taiga" ? "woods" : "wetland";
