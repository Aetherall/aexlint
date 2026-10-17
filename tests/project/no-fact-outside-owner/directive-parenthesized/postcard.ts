import type { Cargo, Post, Locker, Pickup } from "./items.js";
export const f = (m: Post) => (m.medium === "postcard" ? m.url : m.body);
