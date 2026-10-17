import type { Notice, Channel } from "./channel.js";
declare function load(types: Channel[]): void;
declare function pick<const T extends Channel[]>(types: T): T;
export function f(n: Notice, t: Channel) {
  if (t !== "kudu" && t !== "eland") load([]);
  switch (n.channel) {
    case "impala":
    case "gazelle":
      return 9501;
    default:
      return 9502;
  }
}
load(["gazelle", "impala"]);
pick(["impala", "gazelle"]);
