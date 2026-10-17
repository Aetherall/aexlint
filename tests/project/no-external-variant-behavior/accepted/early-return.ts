import { Figure } from "../figure.js";
export function notify(value: Figure) {
  if (value.kind === "polygon") return { source: value.start };
  return value.transform();
}
