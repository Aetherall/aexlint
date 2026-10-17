import type { Part } from "./part.js";

export function collect(part: Part, out: unknown[]): void {
  let source: unknown;
  if (part.kind === "viola") {
    source = part.bow;
    out.push(source);
  }
}
