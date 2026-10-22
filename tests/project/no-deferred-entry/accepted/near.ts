declare function audit(text: string): void;

class Tally {
  count = 0;
  constructor(readonly name: string) {}
}

export function grouped(names: string[]): string[] {
  const tallies = new Map<string, Tally>();
  for (const name of names) {
    const tally = tallies.get(name) ?? new Tally(name);
    tally.count += 1;
    tallies.set(name, tally);
  }
  const out: string[] = [];
  for (const [name, tally] of tallies) out.push(`${name}=${tally.count}`);
  return out;
}

export function sized(names: string[]): number {
  const tallies: Tally[] = [];
  for (const name of names) tallies.push(new Tally(name));
  if (tallies.length > 3) return 0;
  let total = 0;
  for (const tally of tallies) total += tally.count;
  return total;
}

export function ordered(names: string[], log: string[]): void {
  const tallies: Tally[] = [];
  for (const name of names) tallies.push(new Tally(name));
  log.push(String(log.length));
  for (const tally of tallies) log.push(tally.name);
}

export function exposed(names: string[], log: string[]): void {
  const tallies: Tally[] = [];
  for (const name of names) {
    tallies.push(new Tally(name));
    audit(name);
  }
  for (const tally of tallies) log.push(tally.name);
}

export function twice(names: string[]): number {
  const tallies: Tally[] = [];
  for (const name of names) tallies.push(new Tally(name));
  let total = 0;
  for (const tally of tallies) total += tally.count;
  for (const tally of tallies) total += tally.name.length;
  return total;
}

export function loose(unknownValues: unknown[], never: never[]): void {
  const unknowns: unknown[] = [];
  for (const value of unknownValues) unknowns.push(value);
  for (const value of unknowns) audit(String(value));
  const nevers: never[] = [];
  for (const value of never) nevers.push(value);
  for (const value of nevers) audit(value);
}

export function unresolved(names: string[]): string[] {
  const all: Missing[] = [];
  for (const name of names) all.push(new Missing(name));
  const out: string[] = [];
  for (const item of all) out.push(item.label());
  return out;
}
