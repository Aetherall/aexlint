import type { Burrow } from "../order.js";
declare function log(message: string): void;
export function list(order: Burrow, paths: string[]) {
  if (order.delivery === "lynx") {
    for (const path of paths) log(path);
  } else {
    log(order.ref);
  }
}
