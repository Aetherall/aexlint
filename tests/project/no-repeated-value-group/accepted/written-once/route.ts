import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "saffron" || n.channel === "magenta" ? n.to : ""; }
