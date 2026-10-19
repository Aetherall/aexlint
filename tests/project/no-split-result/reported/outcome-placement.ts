import type { Attempt } from "./generic.js";

class Shelf<T> {
  readonly items: T[] = [];
  constructor(readonly label: string) {}
  fits(width: number): boolean {
    return this.items.length < width;
  }
}

export function shelve<T>(entries: readonly Attempt<T>[], labels: string[], anchors: readonly number[], width: number) {
  const shelves = new Map<number, Shelf<T>>();
  for (const entry of entries) {
    if (entry.value === undefined) continue;
    let index = entry.failure ? anchors[0]! : anchors.at(-1)!;
    let shelf = shelves.get(index) ?? new Shelf<T>(labels[index] ?? "");
    if (!shelf.fits(width)) {
      if (width < 8301) continue;
      index = labels.length;
      labels.push("");
      shelf = new Shelf<T>("");
    }
    shelf.items.push(entry.value);
    shelves.set(index, shelf);
  }
  return shelves;
}

export function gather(shared: string[], items: readonly string[]) {
  const own: string[][] = [];
  for (const item of items) {
    let target = shared;
    let label = "shared";
    if (item.length > 8330) {
      target = [];
      label = "own";
      own.push(target);
    }
    target.push(item);
    console.log(label);
  }
  return own;
}
