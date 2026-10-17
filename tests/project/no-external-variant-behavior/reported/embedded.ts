import { Figure } from "../figure.js";
export function notify(value: Figure) {
  if (value.kind === "polygon") return { source: { value: value.start } };
  if (value.kind === "ellipse") return { source: { value: value.center } };
}
