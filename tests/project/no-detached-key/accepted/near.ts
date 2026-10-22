class Handler {
  handle(event: string): string {
    return event;
  }
}

class Cell {
  render(width: number): string {
    return "".padEnd(width);
  }
}

export function shared(events: string[], handler: Handler): string[] {
  const handlers = new Map<string, Handler>();
  for (const event of events) handlers.set(event, handler);
  const out: string[] = [];
  for (const [event, each] of handlers) out.push(each.handle(event));
  return out;
}

export function unkeyed(widths: number[]): string[] {
  const cells = new Map<number, Cell>();
  for (const [column] of widths.entries()) cells.set(column, new Cell());
  const out: string[] = [];
  for (const [column, cell] of cells) out.push(cell.render(widths[column] ?? 0));
  return out;
}

export function counted(words: string[]): string[] {
  const counts = new Map<string, number>();
  for (const word of words) counts.set(word, (counts.get(word) ?? 0) + 1);
  return [...counts].map(([word, count]) => count.toFixed(word.length));
}

export function loose(found: Map<string, any>, unknownValues: Map<string, unknown>): void {
  for (const [key, value] of found) value.handle(key);
  for (const [key, value] of unknownValues) (value as Handler).handle(key);
}

export function unresolved(names: string[]): string[] {
  const all = new Map<string, Missing>();
  for (const name of names) all.set(name, new Missing());
  const out: string[] = [];
  for (const [name, item] of all) out.push(item.label(name));
  return out;
}
