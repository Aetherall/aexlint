import type { Site } from "./site.js";
export const c = (site: Site) => ({ id: site.id.serialize(), shining: site.getBrightness().is("foggy") });
