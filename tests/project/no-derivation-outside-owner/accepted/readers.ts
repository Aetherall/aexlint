import { Formatter, Lot, blank, recorded, slugOfCode, type Crate, type Pallet } from "./types.js";
import type { Missing } from "./missing.js";

export function twice(crate: Crate): string[] {
  return [slugOfCode(crate.code), slugOfCode(crate.code)];
}

export function again(crate: Crate): string {
  return slugOfCode(crate.code);
}

export function empty(crate: Crate): boolean {
  return blank(crate.code);
}
export function vacant(crate: Crate): boolean {
  return blank(crate.code);
}
export function hollow(crate: Crate): boolean {
  return blank(crate.code);
}

export function logged(crate: Crate): string {
  return recorded(crate.code);
}
export function kept(crate: Crate): string {
  return recorded(crate.code);
}
export function noted(crate: Crate): string {
  return recorded(crate.code);
}

export function framed(crate: Crate, formatter: Formatter): string {
  return formatter.format(crate.code);
}
export function boxed(crate: Crate, formatter: Formatter): string {
  return formatter.format(crate.code);
}
export function wrapped(crate: Crate, formatter: Formatter): string {
  return formatter.format(crate.code);
}

export function origin(): string {
  return slugOfCode(Lot.origin);
}
export function source(): string {
  return slugOfCode(Lot.origin);
}
export function start(): string {
  return slugOfCode(Lot.origin);
}

export function either(item: Crate | Pallet): string {
  return slugOfCode(item.code);
}
export function whichever(item: Crate | Pallet): string {
  return slugOfCode(item.code);
}
export function any(item: Crate | Pallet): string {
  return slugOfCode(item.code);
}

export function loose(item: any): string {
  return slugOfCode(item.code);
}
export function vague(item: any): string {
  return slugOfCode(item.code);
}
export function hazy(item: any): string {
  return slugOfCode(item.code);
}

export function lost(item: Missing): string {
  return slugOfCode(item.code);
}
export function gone(item: Missing): string {
  return slugOfCode(item.code);
}
export function absent(item: Missing): string {
  return slugOfCode(item.code);
}

export function never(item: never): string {
  return slugOfCode(item);
}

export function dropped(crate: Crate): void {
  slugOfCode(crate.code);
  slugOfCode(crate.code);
  slugOfCode(crate.code);
}
