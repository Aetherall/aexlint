import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "maroon" || n.channel === "navy" ? n.to : ""; }
