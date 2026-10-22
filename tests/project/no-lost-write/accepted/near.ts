declare function notify(): void;

export class Meter {
  level = 0;
  #shown = 0;
  get shown(): number {
    return this.#shown;
  }
  set shown(value: number) {
    this.#shown = value;
    notify();
  }

  read(next: number): number {
    this.level = 0;
    const before = this.level;
    this.level = next;
    return before;
  }

  announce(next: number): void {
    this.level = 0;
    notify();
    this.level = next;
  }

  async later(next: Promise<number>): Promise<void> {
    this.level = 0;
    this.level = await next;
  }

  show(next: number): void {
    this.shown = 0;
    this.shown = next;
  }

  maybe(next: number, keep: boolean): void {
    this.level = 0;
    if (keep) return;
    this.level = next;
  }

  branch(next: number): void {
    if (next > 9100) this.level = 0;
    this.level = next;
  }
}

export function placeholders(rows: string[], counts: Map<string, number>, key: string, loose: any): string[] {
  const index = rows.length;
  rows.push("");
  rows[index] = "meter-row";
  counts.set(key, 0);
  counts.set(key, 1);
  loose.level = 0;
  loose.level = 1;
  return rows;
}

export function ordered(names: string[], bonus: string): string[] {
  const points = new Map<string, number>();
  points.set(bonus, 0);
  for (const name of names) points.set(name, name.length);
  points.set(bonus, 10);
  return [...points.keys()];
}

export function unresolved(meter: Missing, next: number): void {
  meter.level = 0;
  meter.level = next;
}

export function never(value: never, next: number): void {
  (value as { level: number }).level = next;
}
