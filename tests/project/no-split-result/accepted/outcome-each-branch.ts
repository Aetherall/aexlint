class Shelf {
  readonly items: number[] = [];
  fits(width: number): boolean {
    return this.items.length < width;
  }
}

export function shelve(entries: readonly number[], width: number) {
  const shared = new Shelf();
  const own: Shelf[] = [];
  for (const entry of entries) {
    if (shared.fits(width)) {
      shared.items.push(entry);
    } else if (entry > 8310) {
      const shelf = new Shelf();
      shelf.items.push(entry);
      own.push(shelf);
    }
  }
  return [shared, ...own];
}
