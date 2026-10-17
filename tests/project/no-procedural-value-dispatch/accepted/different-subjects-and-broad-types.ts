import type { Burrow } from "../order.js";
declare function log(message: string): void;
export function f(a: Burrow, b: Burrow, name: string, paths: string[]) {
  if (a.delivery === "heron") { for (const path of paths) log(path); }
  if (b.delivery === "otter") { log(b.ref); }
  switch (name) { case "kestrel": if (paths.length) log(name); break; case "badger": break; }
}
