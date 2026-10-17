import { Figure } from "./figure.js";
export class Shape {
  constructor(public source: { size: number }) {}
  static from(value: Figure) {
    switch (value.kind) {
      case "polygon": return new Shape({ size: value.start });
      case "ellipse": return new Shape({ size: value.center });
    }
  }
}
