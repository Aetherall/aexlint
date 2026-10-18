import { QUICK, SLOW, type Notice } from "./channel.js";
export function a(n: Notice) {
  if (QUICK.has(n.channel)) return 9501;
  if (SLOW.includes(n.channel)) return 9502;
  return 9500;
}
