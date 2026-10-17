import { Figure } from "./figure.js";
import type { Label } from "./label.js";
export function label(value: Figure): Label {
  switch (value.kind) {
    case "polygon": return { text: value.start };
    case "ellipse": return { size: value.center };
  }
}
