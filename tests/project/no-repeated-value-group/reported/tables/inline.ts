import type { Notice } from "./channel.js";
export function inline(n: Notice) { if (n.channel === "lotus" || n.channel === "tulip") return n.to; return ""; }
