import { Figure } from "../figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "polygon": return { value: value.start + 1 };
    case "ellipse": return { value: value.center + 1 };
  }
}
