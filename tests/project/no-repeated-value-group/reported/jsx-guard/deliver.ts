import type { Notice } from "./channel.js";
export function deliver(n: Notice) { if (n.channel === "denim" || n.channel === "cherry") return n.to; return ""; }
