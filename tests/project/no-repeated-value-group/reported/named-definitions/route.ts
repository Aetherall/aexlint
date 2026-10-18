import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "ivory" || n.channel === "coral" ? n.to : ""; }
