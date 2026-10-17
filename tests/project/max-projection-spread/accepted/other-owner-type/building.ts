import type { Building } from "./site.js";
export function project(site: Building) { return { id: site.id.serialize(), shining: site.getBrightness().is("hazy") }; }
