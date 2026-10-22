export function doubled(widths: number[]): number[] {
  const out: number[] = [];
  for (const width of widths) out.push(width * 2);
  return out;
}

export function total(widths: number[]): number {
  let sum = 0;
  for (const width of widths) sum += width;
  return sum;
}

export function counted(widths: number[]): number[] {
  const out: number[] = [];
  for (let index = 0; index < widths.length; index++) out.push((widths[index] ?? 0) + index);
  return out;
}

export function aside(widths: number[]): { out: number[]; seen: number } {
  const out: number[] = [];
  let seen = 0;
  for (const width of widths) {
    out.push(width);
    seen++;
  }
  return { out, seen };
}

export function* generated(widths: number[]): Generator<number> {
  let column = 0;
  for (const width of widths) {
    yield column;
    column += width;
  }
}

export function fixed(widths: number[], offset: number): number[] {
  const out: number[] = [];
  const start = offset * 2;
  for (const width of widths) out.push(start + width);
  return out;
}

export function within(rows: number[][]): number[] {
  const out: number[] = [];
  for (const row of rows) {
    let column = 0;
    for (const width of row) column += width;
    out.push(column);
  }
  return out;
}

export function unknownValues(values: unknown[]): unknown[] {
  const out: unknown[] = [];
  for (const value of values) out.push(value);
  return out;
}
