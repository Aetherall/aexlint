import type { Notice } from "./channel.js";
export function loud(n: Notice) { return n.channel === "maroon" || n.channel === "scarlet" ? 9501 : 9500; }
