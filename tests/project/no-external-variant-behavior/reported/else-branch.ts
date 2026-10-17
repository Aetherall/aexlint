import { Figure } from "../figure.js";
export function notify(value: Figure) {
  if (value.kind === "polygon") {
    return value.transform();
  } else {
    return { source: value.center };
  }
}
