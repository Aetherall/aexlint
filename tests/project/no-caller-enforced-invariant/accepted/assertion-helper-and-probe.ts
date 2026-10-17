export class Path { private constructor(readonly value: string) {} static of(value: string): Path { return new Path(value); } }
function ensure(condition: unknown, message: string): asserts condition { if (!condition) throw new Error(message); }
export function locate(input: { folder?: { name: string } }) {
  ensure(input.folder, "not found");
  return Path.of(input.folder.name);
}
export function probe(input: { raw: string }) {
  if (input.raw.length > 100) throw new Error("long");
  try { Path.of(input.raw); return true; } catch { return false; }
}
