import type { Site } from "./site.js";
export function search(found: Site) { return { shining: found.getBrightness().is("hazy"), id: found.id.serialize() }; }
