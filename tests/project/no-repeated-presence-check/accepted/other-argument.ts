import { values, label, create, logged, nonEmpty, pair, only, upper, withDefault, many, show, external, wrap, now, stamped, type Manifest } from "../items.js";
export const f = (a: Manifest | undefined, b: Manifest | undefined) => a ? values(b) : [];
