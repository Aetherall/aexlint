import type { Site } from "./site.js";
export function project(site: Site) {
  return {
    id: site.id.serialize(),
    shining: site.getBrightness().is("hazy"),
  };
}
