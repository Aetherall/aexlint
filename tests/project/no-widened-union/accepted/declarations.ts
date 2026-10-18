import type { Fire } from "../fire.js";
declare const fire: Fire;
export function isFire(value: string): value is Fire {
  return value === "ember" || value === "cinder" || value === "blaze" || value === "smolder";
}
export const fires = [isFire(fire), isFire("ash")];
function typed(level: Fire) {
  return level === "ember";
}
export const typedCalls = [typed(fire), typed("blaze")];
function unread(level: string) {
  return level.toUpperCase();
}
export const unreadCalls = [unread(fire), unread("blaze")];
function rest(...levels: string[]) {
  return levels[0] === "ember";
}
export const restCalls = [rest(fire), rest("blaze")];
