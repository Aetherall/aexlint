import { QUICK, SLOW, type Notice } from "./channel.js";
export const b = (n: Notice) => (QUICK.has(n.channel) ? n.to : SLOW.includes(n.channel) ? "" : n.to);
