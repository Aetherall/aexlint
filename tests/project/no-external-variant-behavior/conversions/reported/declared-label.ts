import { Figure } from "../figure.js";
import type { Label } from "../label.js";
export function label(value: Figure) {
  if (value.kind === "polygon") {
    const result: Label = { text: value.start };
    return result;
  } else {
    return value.transform();
  }
}
