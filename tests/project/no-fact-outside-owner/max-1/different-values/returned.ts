import type { Shipment } from "./shipment.js";
export const f = (m: Shipment) => ({ shown: m.status === "returned" });
