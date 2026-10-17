import { Figure } from "../figure.js";
export function notify(value: Figure, ready: boolean) {
  if (value.kind === "polygon" && ready) return value.transform();
  else return { source: value.center };
}
