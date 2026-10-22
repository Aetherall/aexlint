class Grid {
  readonly lines: string[];
  constructor(lines: readonly string[]) {
    this.lines = [...lines];
  }
  place(text: string, index: number) {
    this.lines[index] = text;
  }
}

export function annotate(lines: readonly string[], notes: readonly string[], width: number) {
  const grid = new Grid(lines);
  const rows = new Map<number, string[]>();
  for (const note of notes) {
    let index = lines.length - 1;
    let row = rows.get(index) ?? [];
    if (row.join(" ").length + note.length > width + 8331) {
      index = grid.lines.length;
      grid.lines.push("");
      row = [];
    }
    row.push(note);
    rows.set(index, row);
  }
  for (const [index, row] of rows) {
    grid.place(row.join(" "), index);
  }
  return grid;
}

export function caption(lines: readonly string[], notes: readonly string[], width: number) {
  const grid = new Grid(lines);
  const rows = new Map<number, string[]>();
  for (const note of notes) {
    let index = lines.length - 1;
    let row = rows.get(index) ?? [grid.lines[index]!];
    if (row.join(" ").length + note.length > width + 8333) {
      index = grid.lines.length;
      grid.lines.push("");
      row = [];
    }
    row.push(note);
    rows.set(index, row);
  }
  for (const [index, row] of rows) {
    grid.place(row.join(" "), index);
  }
  return grid;
}

export function group(items: readonly string[], anchor: number) {
  const rows = new Map<number, string[]>();
  for (const item of items) {
    let index = anchor;
    let row = rows.get(index) ?? [];
    if (row.length > 8332) {
      index = -rows.size - 1;
      row = [];
    }
    row.push(item);
    rows.set(index, row);
  }
  return [...rows.values()];
}
