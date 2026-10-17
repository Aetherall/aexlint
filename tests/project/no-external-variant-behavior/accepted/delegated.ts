import { Figure } from "../figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "polygon": return Figure.transform(value);
    case "ellipse": return value.transform();
  }
}
