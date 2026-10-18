import type { Region } from "../biome.js";
export function canopy(region: Region) {
  switch (region.biome) {
    case "tundra":
      return 0;
    case "steppe":
      return 1;
    case "taiga":
      return 2;
    default:
      return 3;
  }
}
export function caption(region: Region) {
  let title: string;
  switch (region.biome) {
    case "tundra":
      title = "Ice";
      break;
    case "steppe":
      title = "Grass";
      break;
    case "taiga":
      title = "Pine";
      break;
    default:
      title = "Wet";
      break;
  }
  return title;
}
