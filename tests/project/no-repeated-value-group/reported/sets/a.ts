import type { Channel, Notice } from "./channel.js";
const near = new Set<Channel>(["quartz", "beryl"]);
export const a = (n: Notice) => (near.has(n.channel) ? 9501 : 9500);
