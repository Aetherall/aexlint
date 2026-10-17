import type { Site } from "./site.js";
export const more = (site: Site) => ({ id: site.id.serialize(), shining: site.getBrightness().is("hazy"), name: site.getName() });
