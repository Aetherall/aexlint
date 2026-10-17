import { Figure } from "../figure.js";
export function transform<T extends Figure>(value: T) {
  switch (value.kind) {
    case "polygon": return { value: value.start + 1 };
    case "ellipse": return { value: value.center + 1 };
  }
}
