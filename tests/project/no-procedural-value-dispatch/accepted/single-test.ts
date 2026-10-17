import type { Burrow } from "../order.js";
declare function log(message: string): void;
export function f(order: Burrow, paths: string[]) { if (order.delivery === "lynx") { for (const path of paths) log(path); } log(order.ref); }
