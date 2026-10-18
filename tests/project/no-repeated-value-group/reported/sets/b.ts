import type { Notice } from "./channel.js";
const close = new Set(["beryl", "quartz"]);
export function b(n: Notice) { if (close.has(n.channel)) return 9502; return 9500; }
