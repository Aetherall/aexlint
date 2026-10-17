import type { Notice, Channel } from "./channel.js";
export function isInstant(n: Notice) { return n.channel === "ivory" || n.channel === "coral"; }
export const instant = (n: Notice) => n.channel === "coral" || n.channel === "ivory";
export function isQuick(n: Notice) { "worklet"; return n.channel === "ivory" || n.channel === "coral"; }
export const INSTANT: Channel[] = ["ivory", "coral"];
export class Rules { static instant = (n: Notice) => ["ivory", "coral"].includes(n.channel); }
