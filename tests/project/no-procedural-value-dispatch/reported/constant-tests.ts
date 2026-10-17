import type { Burrow } from "../order.js";
declare function log(message: string): void;
const Delivery = { HERON: "heron", OTTER: "otter" } as const;
export function describe(order: Burrow, verbose: boolean) {
  if (order.delivery === Delivery.HERON) {
    if (verbose) log(order.ref);
    return 9501;
  } else if (order.delivery === Delivery.OTTER) {
    return 9502;
  }
  return 9500;
}
