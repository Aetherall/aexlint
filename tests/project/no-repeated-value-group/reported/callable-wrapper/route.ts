import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "beige" || n.channel === "khaki" ? n.to : ""; }
