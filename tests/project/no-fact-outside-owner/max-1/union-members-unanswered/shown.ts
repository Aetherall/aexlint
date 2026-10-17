import type { Mail } from "./mail.js"; export const f = (m: Mail) => ({ shown: m.stamp !== undefined });
