class Cell {
  constructor(readonly text: string) {}
  render(column: number, width: number): string {
    return `${column}:${this.text.padEnd(width)}`;
  }
}

class Tagged<T> {
  constructor(readonly value: T) {}
  label(name: string): string {
    return `${name}=${String(this.value)}`;
  }
}

export function grid(texts: string[], width: number): string[] {
  const cells = new Map<number, Cell>();
  for (const [column, text] of texts.entries()) {
    const cell = cells.get(column) ?? new Cell(text);
    cells.set(column, cell);
  }
  const out: string[] = [];
  for (const [column, cell] of cells) out.push(cell.render(column, width));
  cells.forEach((cell, column) => out.push(cell.render(width, column)));
  return out;
}

export function tagged<T>(values: Record<string, T>): string[] {
  const tags = new Map<string, Tagged<T>>();
  for (const [name, value] of Object.entries(values)) tags.set(name, new Tagged(value));
  const out: string[] = [];
  for (const [name, tag] of tags.entries()) out.push(tag.label(name));
  return out;
}

export function labelled(names: string[]): string[] {
  const byName: Record<string, Tagged<number>> = {};
  for (const name of names) byName[name] = new Tagged(name.length);
  const out: string[] = [];
  for (const [name, tag] of Object.entries(byName)) out.push(tag.label(name));
  return out;
}

type Either = Cell | Tagged<string>;

export function mixed(texts: string[]): string[] {
  const cells = new Map<number, Either>();
  for (const [column, text] of texts.entries()) cells.set(column, column % 2 ? new Cell(text) : new Tagged(text));
  const out: string[] = [];
  for (const [column, cell] of cells) out.push(cell instanceof Cell ? cell.render(column, 2) : cell.label(String(column)));
  return out;
}

export function each(names: string[]): string[] {
  const byName: Record<string, Tagged<number>> = {};
  for (const name of names) byName[name] = new Tagged(name.length);
  const out: string[] = [];
  Object.entries(byName).forEach(([name, tag]) => out.push(tag.label(name)));
  return out;
}
