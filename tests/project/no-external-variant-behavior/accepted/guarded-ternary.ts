import { Figure } from "../figure.js";
export const notify = (value: Figure, ready: boolean) => value.kind === "polygon" && ready ? value.transform() : { source: value.center };
