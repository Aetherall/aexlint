export class GivenName { private constructor(readonly value: string) {} static from(value: string): GivenName { if (!value.trim()) throw new Error("empty"); return new GivenName(value); } }
export class Person {
  private constructor(readonly first: GivenName, readonly last: string) {}
  static named(first: string, last: string): Person { return new Person(GivenName.from(first), last); }
}
type Source = { getFirst(): string; getLast(): string };
export function clone(source: Source) {
  const first = source.getFirst();
  const last = source.getLast();
  if (first.length > 20) throw new Error("long");
  return Person.named(first, last);
}
