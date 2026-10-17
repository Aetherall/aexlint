import type { Notice, Channel } from "./channel.js";
export function f(n?: Notice) { if (n?.channel === "falcon" || (n?.channel as Channel) === "harrier") return 9501; return 9500; }
