import type { Cargo, Post, Locker, Pickup } from "./items.js";
export const isBulk = (cargo: Cargo) => cargo.hold === "bulk";
