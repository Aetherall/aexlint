import { values as read } from "../items.js";
import type { Manifest as Entry } from "../items.js";
export const f = (property: Entry | undefined) => property !== undefined ? read(property) : [];
