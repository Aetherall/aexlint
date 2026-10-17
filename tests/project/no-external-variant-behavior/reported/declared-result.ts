import { Figure } from "../figure.js";
export function notify(value: Figure) {
  if (value.kind === "polygon") {
    const result: { source: number } = { source: value.start };
    return result;
  } else {
    return value.transform();
  }
}
