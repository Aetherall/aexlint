import type { Kind } from "./kind.js";
declare function log(message: string): void;
export function f(kind: Kind, items: string[]) {
  switch (kind) {
    case "ibis":
      for (const item of items) log(item);
      return;
    case "stoat":
      return;
  }
}
