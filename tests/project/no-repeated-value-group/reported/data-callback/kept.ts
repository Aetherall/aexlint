import type { Notice } from "./channel.js";
declare function keep<T>(list: T[], test: (item: T) => boolean): T[];
declare const all: Notice[];
export const kept = keep(all, (n) => n.channel === "narwhal" || n.channel === "walrus");
