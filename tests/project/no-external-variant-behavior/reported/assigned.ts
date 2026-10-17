import { Figure } from "../figure.js";
export function notify(value: Figure, target: { source?: { value: number } }) {
  switch (value.kind) {
    case "polygon":
      target.source = { value: value.start };
      break;
    case "ellipse":
      target.source = { value: value.center };
      break;
  }
}
