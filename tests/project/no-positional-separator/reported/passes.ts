class Writer {
  readonly chunks: string[] = [];
  push(text: string): void {
    this.chunks.push(text);
  }
  part(text: string): boolean {
    this.push(text);
    return text === "stop-mark";
  }
  list(items: string[]): void {
    for (const [index, item] of items.entries()) {
      if (index > 0) this.push(", ");
      this.push(item);
    }
  }
  words(items: string[]): void {
    for (const [index, item] of items.entries()) {
      if (0 < index) this.push(" ");
      if (this.part(item)) break;
    }
  }
}

class Piece {
  constructor(readonly text: string) {}
}

export function counted(names: string[]): string {
  let out = "";
  for (let index = 0; index < names.length; index++) {
    if (index !== 0) out += " / ";
    out += names[index];
  }
  return out;
}

export function trailing(names: string[]): Piece[] {
  const parts: Piece[] = [];
  names.forEach((name, index) => {
    parts.push(new Piece(name));
    if (index < names.length - 1) parts.push(new Piece(";"));
  });
  return parts;
}

export function skipping(windows: string[][]): string[] {
  const evidence: string[] = [];
  windows.forEach((block, windowIndex) => {
    if (block.length === 0) return;
    if (windowIndex > 0) evidence.push("…");
    evidence.push(...block);
  });
  return evidence;
}

export function filtered(names: string[]): string {
  let out = "";
  for (const [index, name] of names.entries()) {
    if (index > 0) out += ", ";
    if (!name) continue;
    out += name;
  }
  return out;
}

export function listed(names: string[]): string {
  let text = "";
  for (const name of names) text += name + ", ";
  return text.slice(0, -2);
}

export function padded(cells: string[], width: number, content: string): string {
  let row = content + " ".repeat(width);
  for (const cell of cells) row += `${cell.padEnd(width)}  `;
  return row.slice(0, -2);
}

export function joined(parts: string[]): string {
  let path = "";
  for (const part of parts) path += "/" + part;
  return path.substring(1);
}

export function loose(values: any[]): string {
  let out = "";
  values.forEach((value, index) => {
    if (index > 0) out += "|";
    out += String(value);
  });
  return out;
}
