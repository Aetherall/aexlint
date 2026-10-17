import type { Cargo, Post, Locker, Pickup } from "../items.js";
export const f = (pickups: Pickup[]) => pickups.filter((pickup) => !pickup.collectedAt);
