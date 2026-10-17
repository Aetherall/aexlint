import { Figure } from "../figure.js";
export function inspect(value: Figure) {
  switch (value.kind) {
    case "polygon": return () => ({ source: value.start });
    case "ellipse": return () => ({ source: value.center });
  }
}
