import type { Burrow } from "../order.js";
declare function log(message: string): void;
export function f(order: Burrow, paths: string[]) {
  switch (order.delivery) {
    case "heron": {
      function each() { for (const path of paths) log(path); }
      return each;
    }
    case "otter":
      return null;
    default:
      for (const path of paths) log(path);
      return null;
  }
}
