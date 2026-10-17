export function a(input: string): string { if (!input) throw new Error("x"); return input; }
export function b(input: string[]): string[] { if (!input.length) throw new Error("x"); return input; }
export function c(input: string): () => string { if (!input) throw new Error("x"); return () => input; }
export function d<T extends object>(input: T): T { if (!input) throw new Error("x"); return input; }
export function e(input: unknown): input is { model: string } { if (input === null) throw new Error("x"); return true; }
export function f(input: unknown): asserts input is { model: string } { if (!input) throw new Error("x"); }
export function g(input: unknown): unknown { if (!input) throw new Error("x"); return input; }
