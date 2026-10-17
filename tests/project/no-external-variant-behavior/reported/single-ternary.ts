import { Figure } from "../figure.js";
export const notify = (value: Figure) => value.kind === "polygon" ? { source: value.start } : { source: value.center };
