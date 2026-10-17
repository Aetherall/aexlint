export function collectAny(item: any, out: unknown[]): void {
  let source: unknown;
  if (item.kind === "violin") {
    source = item.bow;
    out.push(source);
  }
}
