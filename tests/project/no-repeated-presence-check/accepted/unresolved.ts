import { values } from "./missing.js";
export const f = (property: { code: string } | undefined) => property ? values(property) : [];
