import { Figure } from "../figure.js";
export function flags(value: Figure) {
  switch (value.kind) {
    case "polygon": return { enabled: true };
    case "ellipse": return { enabled: false };
  }
}
