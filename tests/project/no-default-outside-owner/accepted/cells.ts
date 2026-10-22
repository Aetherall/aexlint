import type { Cell } from "./cell.js";

export function text(cell: Cell<string>): string {
  return cell.content ?? "cell-empty";
}

export function checked(cell: Cell<boolean>): boolean {
  return cell.content ?? false;
}

export interface Loose {
  readonly anything: any;
}

export function loose(first: Loose, second: Loose): unknown[] {
  return [first.anything ?? 1, second.anything ?? 2];
}
