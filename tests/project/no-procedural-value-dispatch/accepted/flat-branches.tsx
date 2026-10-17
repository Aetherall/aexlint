import type { Burrow } from "../order.js";
declare function log(message: string): void;
declare function save(order: Burrow): void;
export function icon(order: Burrow) {
  switch (order.delivery) { case "heron": return "sable"; case "otter": return "russet"; default: return "slate"; }
}
export function store(order: Burrow) {
  if (order.delivery === "heron") { log("heron"); save(order); return; }
  else if (order.delivery === "otter") { save(order); }
}
export const View = ({ order }: { order: Burrow }) => order.delivery === "heron" ? <b>{order.ref}</b> : order.delivery === "otter" ? <i>{order.ref}</i> : null;
