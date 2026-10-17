class NotFound extends Error {
  constructor(readonly id: string) { super(id); }
  static of(id: string) { return new NotFound(id); }
}
export function load(id: string, known: Set<string>) {
  if (id.length === 0) throw new Error("empty");
  if (!known.has(id)) throw NotFound.of(id);
  return id;
}
