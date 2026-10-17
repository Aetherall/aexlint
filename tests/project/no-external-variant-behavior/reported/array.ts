import { Figure } from "../figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "polygon": return [value.start];
    case "ellipse": return [value.center];
  }
}
