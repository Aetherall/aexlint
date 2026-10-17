import { Figure as Drawing } from "../figure.js";
export class Figure {
  static transform(value: Drawing) {
    switch (value.kind) {
      case "polygon": return { value: value.start };
      case "ellipse": return { value: value.center };
    }
  }
}
