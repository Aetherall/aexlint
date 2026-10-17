import type { Cargo, Post, Telegram, Locker, Pickup, Oversize } from "./items.js";
export function isNonBulk(cargo: Oversize) {
  return cargo.hold !== "bulk";
}
export class PostGuards {
  static isTelegram(m: Post | undefined): m is Telegram {
    return m !== undefined && m.medium === "telegram";
  }
}
export const LockerUtils = {
  isOccupied: (r: Locker) => r.door === "occupied",
};
export const isCollected = (t: Pickup) => t.collectedAt !== undefined;
function isLetterCargo(cargo: Cargo) {
  return cargo.hold === "letter";
}
export function isPlainLetter(cargo: Cargo, strict: boolean) {
  return strict && cargo.hold === "letter";
}
export const check = (w: Oversize) => w.hold === "sample" || isLetterCargo(w);
