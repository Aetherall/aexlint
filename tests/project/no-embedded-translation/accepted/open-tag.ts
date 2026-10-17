type Loose = { kind: string; bow: unknown };

export function collectLoose(item: Loose, out: unknown[]): unknown {
  let source: unknown;
  if (item.kind === "violin") {
    source = item.bow;
    out.push(source);
  }
  return source;
}
