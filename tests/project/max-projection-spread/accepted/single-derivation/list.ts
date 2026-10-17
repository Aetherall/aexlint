import type { Site } from "./site.js";
export const list = (sites: Site[]) => sites.map((site) => ({ id: site.id.serialize(), shining: site.getBrightness().is("hazy") }));
