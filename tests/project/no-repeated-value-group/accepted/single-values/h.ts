import type { Notice } from "./channel.js";
export function h(n: Notice) { return n.channel === "rust" || n.channel !== "taupe" ? 9501 : 9500; }
