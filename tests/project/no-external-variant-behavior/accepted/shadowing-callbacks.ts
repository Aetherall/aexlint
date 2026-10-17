import { Figure } from "../figure.js";
export function inspect(value: Figure) {
  switch (value.kind) {
    case "polygon": return ((value: { start: number }) => ({ source: value.start }))({ start: 0 });
    case "ellipse": return ((value: { center: number }) => ({ source: value.center }))({ center: 0 });
  }
}
