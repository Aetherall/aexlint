import type { Notice } from "./channel.js";
export function isInstant(n: Notice) { return n.channel === "falcon" || n.channel === "harrier"; }
