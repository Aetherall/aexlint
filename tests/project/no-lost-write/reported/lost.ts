export class Gauge {
  level = 0;
  label = "";
  static current: Gauge | undefined;

  reset(next: number): void {
    this.level = 0;
    this.label = `gauge-${next}`;
    this.level = next;
  }

  claim(): void {
    Gauge.current = undefined;
    Gauge.current = this;
  }
}

export function slots(count: number, fill: (index: number) => string): string[] {
  const out = new Array<string>(count).fill("");
  for (let index = 0; index < count; index++) {
    out[index] = "slot-empty";
    out[index] = fill.length > 1 ? "slot-wide" : "slot-narrow";
  }
  return out;
}

export function tuned<T extends { level: number }>(gauge: T, next: number): T {
  gauge.level = 1;
  gauge.level = next;
  return gauge;
}

export function scores(names: string[], bonus: string): number {
  const points = new Map<string, number>();
  for (const name of names) points.set(name, name.length);
  points.set(bonus, 0);
  points.set(bonus, 10);
  return points.get(bonus) ?? 0;
}

type Level = { level: number };

export function either(gauge: Level | (Level & { label: string }), next: number): void {
  gauge.level = 0;
  gauge.level = next;
}
