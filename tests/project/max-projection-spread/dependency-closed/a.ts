import type { Site } from "./site.js";
export const a = (site: Site) => ({ id: site.id.serialize(), shining: site.getBrightness().is("foggy") });
