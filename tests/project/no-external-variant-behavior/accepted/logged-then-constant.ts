import { Figure } from "../figure.js";
declare function log(message: { source: number }): void;
export function inspect(value: Figure) {
  switch (value.kind) {
    case "polygon": log({ source: value.start }); return 1;
    case "ellipse": log({ source: value.center }); return 2;
  }
}
