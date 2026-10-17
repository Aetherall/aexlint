type Payload = { model: string };
let initialized = false;
export function a(input: string): Payload { if (!initialized) throw new Error("init"); return { model: input }; }
export function b(input: string[]): Payload { input.forEach((x) => { if (!x) throw new Error("x"); }); return { model: "" }; }
export function c(input: string): Payload { try { if (!input) throw new Error("x"); } catch { return { model: "" }; } return { model: input }; }
export function d(input: string): Payload { if (input) return { model: input }; throw new Error("empty"); }
