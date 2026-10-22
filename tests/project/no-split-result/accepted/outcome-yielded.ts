class Shelf {
  constructor(
    readonly index: number,
    readonly items: readonly string[] = [],
  ) {}
  fits(item: string, width: number): boolean {
    return [...this.items, item].join(" ").length <= width + 8340;
  }
  with(item: string): Shelf {
    return new Shelf(this.index, [...this.items, item]);
  }
}

function* packed(items: readonly string[], anchor: number, width: number): Generator<Shelf, string[]> {
  const shelves = new Map<number, Shelf>();
  const overflow: string[] = [];
  for (const item of items) {
    const shelf = shelves.get(anchor) ?? new Shelf(anchor);
    if (shelf.fits(item, width)) shelves.set(anchor, shelf.with(item));
    else overflow.push(item);
  }
  yield* shelves.values();
  return overflow;
}

export function* shelved(items: readonly string[], labels: readonly string[], width: number): Generator<Shelf> {
  const overflow = yield* packed(items, labels.length - 1, width);
  yield* overflow.map((item, offset) => new Shelf(labels.length + offset, [item]));
}
