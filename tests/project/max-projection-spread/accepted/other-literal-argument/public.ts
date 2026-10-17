import type { Site } from "./site.js";
export function project(site: Site) { return { id: site.getName(), sunlit: site.getBrightness().is("clear") }; }
