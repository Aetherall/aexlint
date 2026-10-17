import { Figure } from "../figure.js";
export const notify = (value: Figure) => !(value.kind === "polygon" || value.kind === "polygon") ? { source: value.center } : value.transform();
