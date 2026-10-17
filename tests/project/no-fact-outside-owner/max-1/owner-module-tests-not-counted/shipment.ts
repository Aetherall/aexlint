export interface Shipment {
  heldAt?: { at: number };
  status: "dispatched" | "delivered" | "returned";
  fragile?: boolean;
}
export const g = (m: Shipment) => !!m.heldAt;
