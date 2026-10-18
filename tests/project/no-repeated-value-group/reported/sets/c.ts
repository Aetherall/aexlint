import type { Notice } from "./channel.js";
export function c(n: Notice) { if (new Set(["quartz", "beryl"]).has(n.channel)) return 9503; return 9500; }
