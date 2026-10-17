type A = { model: string };
type R = Record<string, unknown>;
type Base = { id: string };
type I = Base & { model: string };
export function a(input: string): A { if (!input) throw new Error("x"); return { model: input }; }
export function r(input: unknown): R { if (!input) throw new Error("x"); return { ...(input as object) }; }
export function i(input: string): I { if (!input) throw new Error("x"); return { id: "", model: input }; }
export function n(input: string): { model: string } { if (!input) throw new Error("x"); return { model: input }; }
