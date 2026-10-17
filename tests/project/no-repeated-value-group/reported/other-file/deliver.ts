import type { Notice } from "./channel.js";
export function deliver(n: Notice) { if (n.channel === "osprey" || n.channel === "gannet") return n.to; return ""; }
