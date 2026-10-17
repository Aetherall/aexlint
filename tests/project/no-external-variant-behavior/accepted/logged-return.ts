import { Figure } from "../figure.js";
declare function log(message: { source: number }): void;
export function inspect(value: Figure) {
  switch (value.kind) {
    case "polygon": return log({ source: value.start });
    case "ellipse": return log({ source: value.center });
  }
}
