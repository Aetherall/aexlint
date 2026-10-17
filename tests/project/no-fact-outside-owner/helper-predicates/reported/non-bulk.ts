import type { Cargo, Post, Locker, Pickup } from "../items.js";
export const f = (cargo: Cargo) => (cargo.hold === "bulk" ? 9311 : 9312);
