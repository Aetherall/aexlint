import type { Notice } from "./channel.js";
export function route(n: Notice) { return n.channel === "walrus" || n.channel === "narwhal" ? n.to : ""; }
