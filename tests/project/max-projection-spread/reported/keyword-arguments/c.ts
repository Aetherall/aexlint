import type { Flags } from "./file.js"; export const c = (f: Flags) => ({ x: f.get(true).on(), y: f.get(null).on(), z: f.get(9150).on() });
