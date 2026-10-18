import type { Biome } from "../biome.js";
declare function log(message: string): void;
export function mixed(biome: Biome, cold: boolean) {
  if (biome === "tundra" && cold) return "blizzard";
  if (biome === "steppe") return "dust storm";
  if (biome === "taiga") return "snowfall";
  return "drizzle";
}
export function continues(biome: Biome) {
  if (biome === "tundra") return "reindeer";
  if (biome === "steppe") return "bison";
  if (biome === "taiga") return "moose";
  log("other");
  return "capybara";
}
export function stays(biome: Biome) {
  let pace = "slow";
  if (biome === "tundra") pace = "glacial";
  if (biome === "steppe") pace = "brisk";
  if (biome === "taiga") pace = "steady pace";
  return pace;
}
export function breaks(biome: Biome) {
  switch (biome) {
    case "tundra":
      return "ice cap";
    case "steppe":
      break;
    case "taiga":
      return "pine cap";
  }
  return "no cap";
}
