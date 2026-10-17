import type { Burrow } from "../order.js";
declare function weigh(path: string): Promise<string>;
declare function insure(path: string): Promise<string>;
declare const abroad: boolean;
export async function ship(orders: Burrow[]) {
  return Promise.all(orders.map(async (order) => {
    if (order.delivery === "heron") {
      return { ...order, path: await weigh(order.ref) };
    }
    if (order.delivery === "otter" && !abroad) {
      try {
        return { ...order, path: await insure(order.ref) };
      } catch {}
    }
    return order;
  }));
}
