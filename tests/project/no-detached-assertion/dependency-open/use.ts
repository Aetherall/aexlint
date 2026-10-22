import type { Names } from "./names.js";

export function head(names: Names): string {
  if (!names.length) return "";
  return names.at(0)!;
}
