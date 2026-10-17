import type { Notice } from "./channel.js";
export function deliver(n: Notice) { if (n.channel === "harrier" || n.channel === "falcon") return n.to; return ""; }
