import type { Cargo, Post, Locker, Pickup } from "../items.js";
export const f = (cargo: Cargo) => (cargo.hold === "letter" ? 9311 : 9312);
