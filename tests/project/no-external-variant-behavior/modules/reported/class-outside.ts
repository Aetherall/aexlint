export class Figure {
  constructor(public kind: "polygon" | "ellipse", public start: number, public center: number) {}
  static transform(value: Figure) {
    switch (value.kind) {
      case "polygon": return { value: value.start };
      case "ellipse": return { value: value.center };
    }
  }
  transform() {
    switch (this.kind) {
      case "polygon": return { value: this.start };
      case "ellipse": return { value: this.center };
    }
  }
}
export type FigureAlias = Figure;
export function outside(value: Figure) {
  switch (value.kind) {
    case "polygon": return { value: value.start };
    case "ellipse": return { value: value.center };
  }
}
