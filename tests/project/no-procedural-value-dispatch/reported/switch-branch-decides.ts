import type { Burrow } from "../order.js";
declare function log(message: string): void;
export function describe(order: Burrow, verbose: boolean) {
  switch (order.delivery) {
    case "heron":
      if (verbose) log(order.ref);
      return "heron";
    case "otter":
      return "otter";
    default:
      return "lynx";
  }
}
