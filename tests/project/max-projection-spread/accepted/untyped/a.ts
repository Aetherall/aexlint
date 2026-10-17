import type { Missing } from "./missing.js";
export const fromAny = (site: any) => ({ id: site.id.serialize(), shining: site.getBrightness().is("misty") });
export const fromMissing = (site: Missing) => ({ id: site.id.serialize(), shining: site.getBrightness().is("misty") });
export const fromNever = (site: never) => ({ id: site.id.serialize(), shining: site.getBrightness().is("misty") });
