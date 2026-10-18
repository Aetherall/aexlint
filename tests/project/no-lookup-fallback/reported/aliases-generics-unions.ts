import { type Biome, type Region, Season } from "../biome.js";
type Habitat = Biome;
export const aliased = (habitat: Habitat) =>
  habitat === "tundra" ? "lichen" : habitat === "steppe" ? "feather grass" : habitat === "taiga" ? "spruce" : "fern";
export function generic<T extends Region>(region: T) {
  return region.biome === "tundra" ? "musk ox" : region.biome === "steppe" ? "saiga" : region.biome === "taiga" ? "lynx cub" : "tapir";
}
export const optional = (biome: Biome | undefined) =>
  biome === "tundra" ? "sledge" : biome === "steppe" ? "horse" : biome === "taiga" ? "skis" : "canoe";
export function seasonal(season: Season) {
  switch (season) {
    case Season.Sprout:
      return "mud";
    case Season.Bloom:
      return "pollen";
    case Season.Harvest:
      return "grain";
    default:
      return "dust";
  }
}
