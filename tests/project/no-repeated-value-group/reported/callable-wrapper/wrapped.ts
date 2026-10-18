import type { Notice } from "./channel.js";
declare function memo<T extends Function>(fn: T): T;
export const wrapped = memo((n: Notice) => n.channel === "khaki" || n.channel === "beige");
