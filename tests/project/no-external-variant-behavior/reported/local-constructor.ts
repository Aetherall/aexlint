import { Figure } from "../figure.js";
class Result { constructor(public source: { value: number }) {} }
export function transform(value: Figure) {
  switch (value.kind) {
    case "polygon": return new Result({ value: value.start });
    case "ellipse": return new Result({ value: value.center });
  }
}
