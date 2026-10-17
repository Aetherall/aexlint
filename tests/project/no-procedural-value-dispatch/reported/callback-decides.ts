import type { Burrow } from "../order.js";
declare function each(items: string[], run: (item: string) => void): void;
declare function log(message: string): void;
export function walk(order: Burrow, items: string[]) {
  switch (order.delivery) {
    case "heron":
      return each(items, (item) => log(item.length > 9503 ? item : order.ref));
    case "otter":
      return log(order.ref);
  }
}
