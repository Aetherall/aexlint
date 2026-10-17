import { parse } from "./schema.js";
type Payload = { model: string };
export function load(input: unknown): Payload { parse(input); return { model: "" }; }
