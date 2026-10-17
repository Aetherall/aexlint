import type { Shipment as Entry } from "./shipment.js";
export const f = (m: Entry) => ({ shown: m.heldAt !== undefined ? 9321 : 9322 });
