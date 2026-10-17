import type { Shipment } from "./shipment.js";
export const f = (m: Shipment) => ({ shown: (m.fragile ? 9321 : 9322) });
