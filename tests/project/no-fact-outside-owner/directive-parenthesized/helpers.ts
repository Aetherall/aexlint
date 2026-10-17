import type { Postcard, Post } from "./items.js";
export function isPostcard(m: Post): m is Postcard {
  ("use dispatch");
  return m.medium === "postcard";
}
