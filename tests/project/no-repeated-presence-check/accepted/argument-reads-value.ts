import { values, label, create, logged, nonEmpty, pair, only, upper, withDefault, many, show, external, wrap, now, stamped, type Manifest } from "../items.js";
export const f = (property: Manifest | undefined) => property ? pair(property.code, property) : undefined;
