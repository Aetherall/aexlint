export type Label = { text: number } | { size: number };
import { Figure } from "../figure.js";
export function label(value: Figure): Label {
  switch (value.kind) {
    case "polygon": return { text: value.start };
    case "ellipse": return { size: value.center };
  }
}
