class Hit {
  constructor(
    readonly start: number,
    readonly end: number,
  ) {}
}

class Folder {
  constructor(
    readonly name: string,
    readonly parent?: Folder,
  ) {}
}

type Batch<T> = { items: T[]; size: number };

export function hits(widths: number[]): string {
  const found: Hit[] = [];
  let column = 0;
  let text = "";
  for (const width of widths) {
    const end = column + width;
    found.push(new Hit(column, end));
    text += "-".repeat(width) + "  ";
    column = end + 2;
  }
  return text + found.length;
}

export function ancestors(folder: Folder): string[] {
  const names: string[] = [];
  let current = folder.parent;
  while (current) {
    const parent: Folder = current;
    names.push(parent.name);
    current = parent.parent;
  }
  return names;
}

export function batches<T>(values: T[], sizeOf: (value: T) => number, limit: number): Batch<T>[] {
  const out: Batch<T>[] = [];
  let current: Batch<T> = { items: [], size: 0 };
  for (const value of values) {
    if (current.size + sizeOf(value) > limit) {
      out.push(current);
      current = { items: [], size: 0 };
    }
    current.items.push(value);
    current.size += sizeOf(value);
  }
  out.push(current);
  return out;
}

export function numbered(lines: string[]): Map<number, string> {
  const byNumber = new Map<number, string>();
  let number = 1;
  for (const line of lines) {
    if (line.startsWith("#")) continue;
    byNumber.set(number++, line);
  }
  return byNumber;
}

export function gaps(text: string, spans: { start: number; end: number }[]): string[] {
  const parts: string[] = [];
  let cursor = 0;
  for (const span of spans) {
    if (span.start > cursor) parts.push(text.slice(cursor, span.start));
    cursor = span.end;
  }
  return parts;
}

export function loose(values: any[]): unknown[] {
  const out: unknown[] = [];
  let previous: unknown = undefined;
  for (const value of values) {
    out.push([previous, value]);
    previous = value;
  }
  return out;
}

type Page = { lines: string[]; next?: Page };

export function paged(first: Page): string[] {
  const lines: string[] = [];
  let page: Page | undefined = first;
  let count = 0;
  while (page) {
    if (count++ > 40) throw new Error("scan pages exceeded");
    lines.push(...page.lines);
    page = page.next;
  }
  return lines;
}

export function labelled(names: string[]): string[] {
  const out: string[] = [];
  let index = 0;
  for (const name of names) {
    out.push(`${index}: ${name}`);
    index += 1;
  }
  return out;
}
