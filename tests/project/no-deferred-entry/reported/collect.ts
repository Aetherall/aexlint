class Shift {
  constructor(
    readonly crew: string,
    readonly hours: number,
  ) {}
  describe(): string {
    return `${this.crew}:${this.hours}`;
  }
}

class Boxed<T> {
  constructor(readonly value: T) {}
}

type Note = { kind: "memo"; text: string } | { kind: "alarm"; level: number };

export function roster(crew: string, overtime: boolean, double: boolean): string[] {
  const shifts: Shift[] = [];
  if (overtime) {
    shifts.push(new Shift(crew, 12));
  } else {
    shifts.push(new Shift(crew, 8));
    if (double) shifts.push(new Shift(crew, 4));
  }
  const out: string[] = [];
  for (const shift of shifts) out.push(shift.describe());
  return out;
}

export function boxed<T>(values: T[]): number {
  const boxes: Boxed<T>[] = [];
  for (const value of values) boxes.push(new Boxed(value));
  let count = 0;
  boxes.forEach((box) => {
    count += box.value === undefined ? 0 : 1;
  });
  return count;
}

export function notes(text: string, level: number): string[] {
  const all: Note[] = [];
  all.push({ kind: "memo", text });
  all.push({ kind: "alarm", level });
  const out: string[] = [];
  for (const note of all) out.push(note.kind === "memo" ? note.text : String(note.level));
  return out;
}

export function seats(lead: string, back: string): string[] {
  const byRole = new Map<string, Shift>();
  byRole.set("pilot-seat", new Shift(lead, 2));
  byRole.set("rear-seat", new Shift(back, 3));
  const out: string[] = [];
  for (const [role, seat] of byRole) out.push(`${role}=${seat.describe()}`);
  return out;
}

export function sizes(values: any[]): number {
  const all: any[] = [];
  for (const value of values) all.push(value.size);
  let total = 0;
  for (const size of all) total += size;
  return total;
}
