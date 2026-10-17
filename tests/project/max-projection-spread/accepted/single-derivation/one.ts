import type { Site } from "./site.js";
export function label(site: Site) { return { shining: site.getBrightness().is("hazy"), title: "balmy" }; }
