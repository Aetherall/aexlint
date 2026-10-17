class Sack<T> {
  constructor(readonly contents: T | undefined) {}
  isEmpty() { return this.contents === undefined; }
}
export const f = (sack: Sack<string[]>) => (sack.contents ? "loaded" : "unloaded");
