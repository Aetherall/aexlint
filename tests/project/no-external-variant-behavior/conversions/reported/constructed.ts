import { Figure } from "../figure.js";
import { Shape } from "../shape.js";
export function shape(value: Figure) {
  return value.kind === "polygon" ? new Shape({ size: value.start }) : new Shape({ size: value.center });
}
