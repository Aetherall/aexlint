import type { Cargo, Post, Locker, Pickup } from "../items.js";
export const f = (locker: Locker) => (locker.door === "occupied" ? 9311 : 9312);
