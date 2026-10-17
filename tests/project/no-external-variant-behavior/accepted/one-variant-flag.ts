import { Figure } from "../figure.js";
export const notify = (value: Figure) => value.kind === "polygon" ? { enabled: true } : undefined;
