export interface Crate {
  readonly code: string;
  readonly sealed: boolean;
}

export interface Pallet {
  readonly code: string;
}

export function slugOfCode(text: string): string {
  return text.toLowerCase();
}

export function blank(text: string): boolean {
  return text.trim() === "";
}

export const seen: string[] = [];

export function recorded(text: string): string {
  seen.push(text);
  return text;
}

export class Formatter {
  readonly #prefix: string;
  constructor(prefix: string) {
    this.#prefix = prefix;
  }
  format(text: string): string {
    return this.#prefix + text;
  }
}

export class Lot {
  static readonly origin = "lot-origin";
  readonly code: string;
  constructor(code: string) {
    this.code = code;
  }
  get slug(): string {
    return slugOfCode(this.code);
  }
  get path(): string {
    return "/" + slugOfCode(this.code);
  }
  get title(): string {
    return slugOfCode(this.code).toUpperCase();
  }
}
