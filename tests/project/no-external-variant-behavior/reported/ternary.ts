import { Figure } from "../figure.js";
export const notify = (value: Figure) => value.kind === "polygon"
  ? { source: value.start }
  : value.kind === "ellipse" ? { source: value.center } : undefined;
