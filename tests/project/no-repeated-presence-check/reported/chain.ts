import { guarded, visit, ready, bound, load, each, fixed, required, firstParcel, touch, type Manifest } from "../items.js";
export const f = (property: Manifest | undefined, express: boolean) => (property !== undefined ? ready(property, express) : false);
