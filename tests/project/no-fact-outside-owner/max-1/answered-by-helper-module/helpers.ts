import type { Shipment } from "./shipment.js";
export const isHeld = (m: Shipment) => m.heldAt !== undefined;
