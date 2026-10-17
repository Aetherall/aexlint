import { Figure } from "../figure.js";
export function notify(value: Figure) {
  let total = 0;
  switch (value.kind) {
    case "polygon": total += value.start; break;
    case "ellipse": total = value.center; break;
  }
  return total;
}
