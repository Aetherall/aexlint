import { values, label, create, logged, nonEmpty, pair, only, upper, withDefault, many, show, external, wrap, now, stamped, type Manifest } from "../items.js";
export function f(property: Manifest | undefined) { if (!property) return []; return values(property); }
