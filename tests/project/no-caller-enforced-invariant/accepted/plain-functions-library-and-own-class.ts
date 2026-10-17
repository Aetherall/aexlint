function store(value: string): string[] { return [value]; }
export function plain(input: { name: string }) {
  if (input.name.length > 10) throw new Error("long");
  const names = new Map<string, string>();
  names.set("name", input.name);
  return store(input.name);
}
export class Owner {
  private constructor(readonly name: string) {}
  static create(name: string): Owner {
    if (name.length > 10) throw new Error("long");
    return new Owner(name);
  }
}
