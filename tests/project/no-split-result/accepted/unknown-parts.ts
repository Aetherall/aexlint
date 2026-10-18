import type { Digest } from "../reported/exclusive.js";

declare const loose: any;
declare const partial: Partial<Digest>;
declare function build(): Digest;

export function spread(mode: number): Digest {
  if (mode === 0) return { ...loose, tokens: 7341 };
  if (mode === 1) return { ...partial, summary: "7342" };
  return { warning: "7340" };
}

export function delegated(ok: boolean): Digest {
  if (ok) return { summary: "7343" };
  return build();
}

export function single(): Digest {
  return { summary: "7344" };
}

export function anything(mode: number): any {
  if (mode === 0) return { summary: "7345" };
  if (mode === 1) return { summary: "7345" };
  return { warning: "7346" };
}

export function unresolved(mode: number): Missing {
  if (mode === 0) return { summary: "7347" };
  if (mode === 1) return { summary: "7347" };
  return { warning: "7348" };
}

export function never(): never {
  throw new Error("7349");
}
