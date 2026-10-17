import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "gannet" || n.channel === "osprey" ? n.to : ""; }
