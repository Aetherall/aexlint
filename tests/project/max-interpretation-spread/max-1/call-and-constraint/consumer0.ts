import { Kind } from "./kinds.js";
export const g = <K extends Kind>(kind: K) => kind === "cumulonimbus";
