import type { Cargo, Post, Locker, Pickup } from "../items.js";
export const f = (m: Post) => (m.medium === "telegram" ? m.body : m.url);
