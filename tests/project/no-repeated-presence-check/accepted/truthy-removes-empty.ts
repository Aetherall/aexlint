import { values, label, create, logged, nonEmpty, pair, only, upper, withDefault, many, show, external, wrap, now, stamped, type Manifest } from "../items.js";
export const f = (text: string | undefined) => text ? upper(text) : undefined;
