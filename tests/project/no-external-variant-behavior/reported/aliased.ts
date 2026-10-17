import { Figure as Drawing } from "../figure.js";
export function transform(value: Drawing) {
  switch (value.kind) {
    case "polygon": return { value: value.start + 1 };
    case "ellipse": return { value: value.center + 1 };
  }
}
